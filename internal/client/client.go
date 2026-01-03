package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/sombochea/tungo/internal/client/introspect"
	"github.com/sombochea/tungo/pkg/config"
	"github.com/sombochea/tungo/pkg/protocol"
	"github.com/sombochea/tungo/pkg/version"
)

// Buffer pool for high-performance data forwarding
var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024) // 32KB buffers
		return &buf
	},
}

// TunnelClient represents a tunnel client
type TunnelClient struct {
	config           *config.ClientConfig
	logger           zerolog.Logger
	conn             *websocket.Conn
	connMutex        sync.Mutex
	streams          map[protocol.StreamID]*LocalStream
	streamMux        sync.RWMutex
	send             chan []byte
	done             chan struct{}
	closed           bool
	closeMutex       sync.Mutex
	serverInfo       *protocol.ServerHello
	currentServerIdx int // Current server index in cluster
	serverList       []config.ServerNode
}

// LocalStream represents a connection to the local server
type LocalStream struct {
	ID             protocol.StreamID
	LocalConn      net.Conn
	DataChan       chan []byte
	Done           chan struct{}
	RequestWritten chan struct{} // Signal when request has been written
	BytesSent      int64
	BytesRecv      int64
	RequestData    []byte // Capture request for introspect
	ResponseData   []byte // Capture response for introspect
	captureEnabled bool
	StartTime      time.Time // Track request start time
	EndTime        time.Time // Track response end time
	Method         string    // HTTP method
	Path           string    // HTTP path
	SourceIP       string    // Client source IP
	StatusCode     int       // HTTP status code
	firstRead      bool      // Track if we've done first read
}

// NewTunnelClient creates a new tunnel client
func NewTunnelClient(cfg *config.ClientConfig, logger zerolog.Logger) *TunnelClient {
	return &TunnelClient{
		config:           cfg,
		logger:           logger,
		streams:          make(map[protocol.StreamID]*LocalStream),
		send:             make(chan []byte, 256),
		done:             make(chan struct{}),
		currentServerIdx: 0,
		serverList:       cfg.GetServerList(), // Get server list from config
	}
}

// Connect establishes a connection to the tunnel server
func (tc *TunnelClient) Connect() error {
	tc.connMutex.Lock()
	defer tc.connMutex.Unlock()

	// Close existing connection and wait for cleanup
	if tc.conn != nil {
		tc.logger.Debug().Msg("Closing old connection and waiting for goroutines to finish")

		// Close the old connection
		tc.conn.Close()

		// Close done channel to signal goroutines to stop
		tc.closeMutex.Lock()
		if !tc.closed {
			tc.closed = true
			select {
			case <-tc.done:
			default:
				close(tc.done)
			}
		}
		tc.closeMutex.Unlock()

		// Wait for goroutines to finish
		time.Sleep(500 * time.Millisecond)
	}

	// Reset closed flag for new connection
	tc.closeMutex.Lock()
	tc.closed = false
	tc.closeMutex.Unlock()

	// Clean up streams
	tc.streamMux.Lock()
	for _, stream := range tc.streams {
		select {
		case <-stream.Done:
		default:
			close(stream.Done)
		}
		stream.LocalConn.Close()
	}
	tc.streams = make(map[protocol.StreamID]*LocalStream)
	tc.streamMux.Unlock()

	// Create fresh channels for new connection
	tc.send = make(chan []byte, 256)
	tc.done = make(chan struct{})

	// Note: We preserve tc.serverInfo to reuse subdomain on reconnection

	// Get current server from cluster
	currentServer := tc.serverList[tc.currentServerIdx]

	// Build WebSocket URL with appropriate scheme
	scheme := "ws"
	if currentServer.Secure {
		scheme = "wss"
	}

	wsURL := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", currentServer.Host, currentServer.Port),
		Path:   "/ws",
	}

	tc.logger.Info().
		Str("url", wsURL.String()).
		Int("server_index", tc.currentServerIdx).
		Int("total_servers", len(tc.serverList)).
		Msg("Connecting to server")

	// Configure WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: tc.config.ConnectTimeout,
	}

	// Configure TLS if using secure connection
	if currentServer.Secure {
		if tc.config.InsecureTLS {
			// Skip TLS verification (for testing only)
			dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			tc.logger.Warn().Msg("TLS certificate verification disabled (insecure mode)")
		}
		// Otherwise use default TLS config which validates certificates
	}

	// Set request headers
	headers := make(map[string][]string)
	headers["User-Agent"] = []string{fmt.Sprintf("TunGo-Client/%s", version.GetShortVersion())}
	
	// For Cloudflare and standard HTTPS/WSS ports, use clean Host header without port
	if currentServer.Secure && currentServer.Port == 443 {
		headers["Host"] = []string{currentServer.Host}
	}

	// Connect to WebSocket
	conn, resp, err := dialer.Dial(wsURL.String(), headers)
	if err != nil {
		// Log detailed error information
		if resp != nil {
			tc.logger.Error().
				Int("status_code", resp.StatusCode).
				Str("status", resp.Status).
				Msg("WebSocket handshake failed")
		}
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	tc.conn = conn

	// Send client hello
	if err := tc.sendClientHello(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send client hello: %w", err)
	}

	// Receive server hello
	if err := tc.receiveServerHello(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive server hello: %w", err)
	}

	tc.logger.Info().
		Str("subdomain", tc.serverInfo.SubDomain).
		Str("hostname", tc.serverInfo.Hostname).
		Msg("Tunnel established")

	return nil
}

// sendClientHello sends the initial hello message to the server
func (tc *TunnelClient) sendClientHello() error {
	var hello *protocol.ClientHello

	if tc.config.ReconnectToken != "" {
		// Reconnecting with token
		hello = protocol.NewReconnectHello(&protocol.ReconnectToken{
			Token: tc.config.ReconnectToken,
		})
	} else {
		// New connection or reconnection
		var subDomain *string

		// First check if we have a subdomain from previous connection
		if tc.serverInfo != nil && tc.serverInfo.SubDomain != "" {
			subDomain = &tc.serverInfo.SubDomain
			tc.logger.Debug().Str("subdomain", *subDomain).Msg("Reusing subdomain from previous session")
		} else if tc.config.SubDomain != "" {
			// Use configured subdomain
			subDomain = &tc.config.SubDomain
		}

		var secretKey *protocol.SecretKey
		if tc.config.SecretKey != "" {
			secretKey = &protocol.SecretKey{
				Key: tc.config.SecretKey,
			}
		}

		hello = protocol.NewClientHello(subDomain, secretKey)
	}

	// Set client version
	hello.SetClientVersion(version.GetShortVersion())

	return tc.conn.WriteJSON(hello)
}

// receiveServerHello receives the server hello response
func (tc *TunnelClient) receiveServerHello() error {
	var hello protocol.ServerHello
	if err := tc.conn.ReadJSON(&hello); err != nil {
		return fmt.Errorf("failed to read server hello: %w", err)
	}

	if hello.Type != protocol.ServerHelloSuccess {
		return fmt.Errorf("server rejected connection: %s - %s", hello.Type, hello.Error)
	}

	tc.serverInfo = &hello
	return nil
}

// Run starts the client's main event loop
func (tc *TunnelClient) Run() error {
	tc.logger.Info().Msg("Client event loop started")

	// Start read and write pumps
	go tc.writePump()
	go tc.readPump()

	// Wait for done signal
	<-tc.done

	tc.logger.Info().Msg("Client event loop ended")
	return nil
}

// readPump reads messages from the WebSocket connection
func (tc *TunnelClient) readPump() {
	defer func() {
		tc.logger.Info().Msg("readPump stopped")
		// Signal that connection is broken
		tc.closeMutex.Lock()
		if !tc.closed {
			tc.closed = true
			close(tc.done)
		}
		tc.closeMutex.Unlock()
	}()

	tc.logger.Info().Msg("readPump started")

	for {
		var msg protocol.Message
		tc.logger.Debug().Msg("Waiting to read WebSocket message...")
		err := tc.conn.ReadJSON(&msg)
		if err != nil {
			// Log the actual error with full details
			tc.logger.Error().
				Err(err).
				Str("error_type", fmt.Sprintf("%T", err)).
				Bool("is_unexpected", websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure)).
				Msg("WebSocket ReadJSON error")
			return
		}

		tc.logger.Debug().Str("type", string(msg.Type)).Msg("Received message")
		tc.handleMessage(&msg)
	}
}

// writePump writes messages to the WebSocket connection
func (tc *TunnelClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer tc.logger.Info().Msg("writePump stopped")

	tc.logger.Info().Msg("writePump started")

	for {
		select {
		case message, ok := <-tc.send:
			if !ok {
				tc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := tc.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				tc.logger.Warn().Err(err).Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			// Send pong in response to ping
			pongMsg, _ := protocol.NewMessage(protocol.MessageTypePong, "", nil)
			data, _ := protocol.EncodeMessage(pongMsg)
			if err := tc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				tc.logger.Debug().Err(err).Msg("Failed to send pong")
				return
			}

		case <-tc.done:
			return
		}
	}
}

// handleMessage handles a received message

// handleMessage handles an incoming message
func (tc *TunnelClient) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypePing:
		// Respond with pong
		pongMsg, _ := protocol.NewMessage(protocol.MessageTypePong, "", nil)
		data, _ := protocol.EncodeMessage(pongMsg)
		select {
		case tc.send <- data:
		default:
			tc.logger.Warn().Msg("Send buffer full, dropping pong")
		}

	case protocol.MessageTypeInit:
		// Initialize new stream
		var initMsg protocol.InitStreamMessage
		if err := msg.Unmarshal(&initMsg); err != nil {
			tc.logger.Error().Err(err).Msg("Failed to unmarshal init message")
			return
		}
		tc.handleInitStream(&initMsg)

	case protocol.MessageTypeData:
		// Forward data to local stream
		stream, exists := tc.getStream(msg.StreamID)
		if !exists {
			tc.logger.Warn().Str("stream_id", msg.StreamID.String()).Msg("Stream not found for data message")
			return
		}

		var dataMsg protocol.DataMessage
		if err := msg.Unmarshal(&dataMsg); err != nil {
			tc.logger.Error().Err(err).Msg("Failed to unmarshal data message")
			return
		}

		select {
		case stream.DataChan <- dataMsg.Data:
		case <-stream.Done:
			tc.logger.Debug().Str("stream_id", msg.StreamID.String()).Msg("Stream closed while sending data")
		default:
			tc.logger.Warn().Str("stream_id", msg.StreamID.String()).Msg("Stream data channel full")
		}

	case protocol.MessageTypeEnd:
		// Close stream
		tc.logger.Debug().Str("stream_id", msg.StreamID.String()).Msg("Received stream end")
		tc.closeStream(msg.StreamID)

	default:
		tc.logger.Warn().Str("type", string(msg.Type)).Msg("Unknown message type")
	}
}

// handleInitStream handles a stream initialization message
func (tc *TunnelClient) handleInitStream(initMsg *protocol.InitStreamMessage) {
	tc.logger.Debug().
		Str("stream_id", initMsg.StreamID.String()).
		Str("protocol", initMsg.Protocol).
		Msg("Initializing new stream")

	// Connect to local server
	localAddr := net.JoinHostPort(tc.config.LocalHost, fmt.Sprintf("%d", tc.config.LocalPort))
	localConn, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
	if err != nil {
		tc.logger.Error().Err(err).Msg("Failed to connect to local server")
		tc.sendStreamEnd(initMsg.StreamID)
		return
	}

	// Create stream with larger buffer for high throughput
	stream := &LocalStream{
		ID:             initMsg.StreamID,
		LocalConn:      localConn,
		DataChan:       make(chan []byte, 512), // Increased from 256 for better throughput
		Done:           make(chan struct{}),
		RequestWritten: make(chan struct{}), // Signal channel
		captureEnabled: tc.config.EnableDashboard,
		StartTime:      time.Now(), // Record start time
	}

	tc.addStream(stream)

	// Start both proxy goroutines
	// proxyToLocal will write request data, then signal proxyFromLocal to read response
	go tc.proxyToLocal(stream)
	go tc.proxyFromLocal(stream)
}

// proxyToLocal forwards data from the tunnel to the local server
func (tc *TunnelClient) proxyToLocal(stream *LocalStream) {
	defer func() {
		tc.logger.Debug().Str("stream_id", stream.ID.String()).Msg("proxyToLocal finished")
	}()

	requestComplete := false

	for {
		select {
		case data, ok := <-stream.DataChan:
			if !ok {
				return
			}

			// Parse request on first data chunk (but don't log yet - wait for response)
			if !requestComplete && len(data) > 0 {
				// Parse HTTP request line
				dataStr := string(data)
				if len(dataStr) > 0 {
					lines := make([]string, 0)
					lastIdx := 0
					for i := 0; i < len(dataStr); i++ {
						if dataStr[i] == '\n' {
							lines = append(lines, dataStr[lastIdx:i])
							lastIdx = i + 1
							if len(lines) >= 20 { // Parse first 20 lines for headers
								break
							}
						}
					}

					// Parse request line (first line)
					if len(lines) > 0 {
						parts := make([]string, 0)
						lastIdx := 0
						for i := 0; i < len(lines[0]); i++ {
							if lines[0][i] == ' ' {
								if i > lastIdx {
									parts = append(parts, lines[0][lastIdx:i])
								}
								lastIdx = i + 1
							}
						}
						if lastIdx < len(lines[0]) {
							parts = append(parts, lines[0][lastIdx:])
						}

						if len(parts) >= 2 {
							stream.Method = parts[0]
							stream.Path = parts[1]
						}
					}

					// Parse headers to find X-Forwarded-For or X-Real-IP
					for i := 1; i < len(lines); i++ {
						line := lines[i]
						if len(line) > 16 && (line[:16] == "X-Forwarded-For:" || line[:16] == "x-forwarded-for:") {
							stream.SourceIP = line[17:]
							// Trim carriage return if present
							if len(stream.SourceIP) > 0 && stream.SourceIP[len(stream.SourceIP)-1] == '\r' {
								stream.SourceIP = stream.SourceIP[:len(stream.SourceIP)-1]
							}
							break
						} else if len(line) > 11 && (line[:11] == "X-Real-IP: " || line[:11] == "x-real-ip: ") {
							stream.SourceIP = line[11:]
							if len(stream.SourceIP) > 0 && stream.SourceIP[len(stream.SourceIP)-1] == '\r' {
								stream.SourceIP = stream.SourceIP[:len(stream.SourceIP)-1]
							}
							break
						}
					}
				}
			}

			// Capture request data if dashboard is enabled
			if stream.captureEnabled {
				stream.RequestData = append(stream.RequestData, data...)
			}

			// Write data to local server
			n, err := stream.LocalConn.Write(data)
			if err != nil {
				tc.logger.Debug().Err(err).Str("stream_id", stream.ID.String()).Msg("Failed to write to local server")
				return
			}
			stream.BytesSent += int64(n)

			// After first write, signal that request has been written
			if !requestComplete {
				requestComplete = true
				close(stream.RequestWritten) // Signal immediately after first write
				tc.logger.Debug().Str("stream_id", stream.ID.String()).Int("bytes", n).Msg("HTTP request written to local server, signaling reader")
			}

		case <-stream.Done:
			return
		}
	}
}

// proxyFromLocal forwards data from the local server to the tunnel
func (tc *TunnelClient) proxyFromLocal(stream *LocalStream) {
	defer func() {
		// Log complete request/response in standard format
		if stream.StatusCode > 0 && stream.Method != "" {
			// Use EndTime if set, otherwise use current time
			endTime := stream.EndTime
			if endTime.IsZero() {
				endTime = time.Now()
			}
			latency := endTime.Sub(stream.StartTime)
			timestamp := stream.StartTime.Format("2006/01/02 15:04:05")
			sourceIP := stream.SourceIP
			if sourceIP == "" {
				sourceIP = "-"
			}

			// Color code status
			statusColor := ""
			resetColor := ""
			if stream.StatusCode >= 200 && stream.StatusCode < 300 {
				statusColor = "\033[32m" // Green
				resetColor = "\033[0m"
			} else if stream.StatusCode >= 300 && stream.StatusCode < 400 {
				statusColor = "\033[36m" // Cyan
				resetColor = "\033[0m"
			} else if stream.StatusCode >= 400 && stream.StatusCode < 500 {
				statusColor = "\033[33m" // Yellow
				resetColor = "\033[0m"
			} else if stream.StatusCode >= 500 {
				statusColor = "\033[31m" // Red
				resetColor = "\033[0m"
			}

			// Format: [timestamp] source_ip "METHOD /path" status req_bytes res_bytes latency_ms
			fmt.Printf("%s %s \"%s %s\" %s%d%s %d %d %dms\n",
				timestamp, sourceIP, stream.Method, stream.Path,
				statusColor, stream.StatusCode, resetColor,
				stream.BytesSent, stream.BytesRecv, latency.Milliseconds())
		}

		// Capture the request/response if dashboard is enabled
		if stream.captureEnabled && len(stream.RequestData) > 0 {
			introspect.CaptureStream(stream.RequestData, stream.ResponseData)
		}

		tc.sendStreamEnd(stream.ID)
		tc.closeStream(stream.ID)
	}()

	// Wait for request to be written before reading response
	tc.logger.Debug().Str("stream_id", stream.ID.String()).Msg("Waiting for request to be written...")
	<-stream.RequestWritten

	// Add small delay to ensure local server has processed the request
	time.Sleep(10 * time.Millisecond)

	tc.logger.Debug().Str("stream_id", stream.ID.String()).Msg("Request written, starting to read response")

	// Get buffer from pool for high performance
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufferPool.Put(bufPtr)

	for {
		select {
		case <-stream.Done:
			return
		default:
			// Set read deadline to avoid blocking forever
			// Use shorter timeout after first read for better responsiveness
			timeout := 5 * time.Second
			if stream.firstRead {
				timeout = 500 * time.Millisecond // Shorter timeout after we've started reading
			}
			stream.LocalConn.SetReadDeadline(time.Now().Add(timeout))

			n, err := stream.LocalConn.Read(buf)
			if err != nil {
				// Check if it's a timeout (expected) or real error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout means no more data
					if stream.BytesRecv > 0 {
						// We've received data, mark end time and finish
						stream.EndTime = time.Now()
						tc.logger.Debug().Str("stream_id", stream.ID.String()).Msg("Read timeout, response complete")
						return
					}
					// No data has been received yet, keep waiting
					continue
				}
				if err == io.EOF {
					// Normal end of response
					stream.EndTime = time.Now()
					tc.logger.Debug().Str("stream_id", stream.ID.String()).Msg("EOF received, response complete")
				} else {
					tc.logger.Debug().Err(err).Str("stream_id", stream.ID.String()).Msg("Local connection closed")
				}
				return
			}

			if n > 0 {
				if !stream.firstRead {
					stream.firstRead = true
				}
				stream.BytesRecv += int64(n)

				// Capture response data if dashboard is enabled
				if stream.captureEnabled {
					stream.ResponseData = append(stream.ResponseData, buf[:n]...)
				}

				// Parse and log HTTP response status on first read
				if stream.BytesRecv == int64(n) && n > 12 {
					// This is the first chunk, try to extract status code
					statusLine := string(buf[:n])
					if len(statusLine) > 12 && statusLine[:5] == "HTTP/" {
						// Find the end of the status line
						endIdx := 0
						for i := 0; i < len(statusLine) && i < 100; i++ {
							if statusLine[i] == '\n' {
								endIdx = i
								break
							}
						}
						if endIdx > 0 {
							// Parse status code
							parts := make([]string, 0)
							lastIdx := 0
							line := statusLine[:endIdx]
							for i := 0; i < len(line); i++ {
								if line[i] == ' ' {
									if i > lastIdx {
										parts = append(parts, line[lastIdx:i])
									}
									lastIdx = i + 1
								}
							}
							if lastIdx < len(line) {
								parts = append(parts, line[lastIdx:])
							}

							if len(parts) >= 2 {
								// Parse status code
								statusCode := 0
								for i := 0; i < len(parts[1]); i++ {
									if parts[1][i] >= '0' && parts[1][i] <= '9' {
										statusCode = statusCode*10 + int(parts[1][i]-'0')
									}
								}
								stream.StatusCode = statusCode
							}
						}
					}
				}

				tc.logger.Debug().
					Str("stream_id", stream.ID.String()).
					Int("bytes_read", n).
					Msg("Read from local server")

				// Send data through tunnel - copy buffer to avoid data race
				dataMsg := &protocol.DataMessage{
					Data: append([]byte(nil), buf[:n]...), // Copy the buffer
				}
				msg, err := protocol.NewMessage(protocol.MessageTypeData, stream.ID, dataMsg)
				if err != nil {
					tc.logger.Error().Err(err).Msg("Failed to create data message")
					return
				}

				data, err := protocol.EncodeMessage(msg)
				if err != nil {
					tc.logger.Error().Err(err).Msg("Failed to encode message")
					return
				}

				select {
				case tc.send <- data:
				case <-stream.Done:
					return
				case <-time.After(5 * time.Second):
					tc.logger.Warn().Str("stream_id", stream.ID.String()).Msg("Send buffer full, timing out")
					return
				}
			}
		}
	}
}

// sendStreamEnd sends a stream end message
func (tc *TunnelClient) sendStreamEnd(streamID protocol.StreamID) {
	msg, _ := protocol.NewMessage(protocol.MessageTypeEnd, streamID, nil)
	data, _ := protocol.EncodeMessage(msg)

	select {
	case tc.send <- data:
	case <-tc.done:
	default:
		tc.logger.Warn().Str("stream_id", streamID.String()).Msg("Failed to send stream end")
	}
}

// addStream adds a stream to the client
func (tc *TunnelClient) addStream(stream *LocalStream) {
	tc.streamMux.Lock()
	defer tc.streamMux.Unlock()
	tc.streams[stream.ID] = stream
}

// getStream retrieves a stream by ID
func (tc *TunnelClient) getStream(streamID protocol.StreamID) (*LocalStream, bool) {
	tc.streamMux.RLock()
	defer tc.streamMux.RUnlock()
	stream, exists := tc.streams[streamID]
	return stream, exists
}

// closeStream closes a stream
func (tc *TunnelClient) closeStream(streamID protocol.StreamID) {
	tc.streamMux.Lock()
	defer tc.streamMux.Unlock()

	stream, exists := tc.streams[streamID]
	if !exists {
		return
	}

	close(stream.Done)
	stream.LocalConn.Close()
	delete(tc.streams, streamID)

	tc.logger.Debug().
		Str("stream_id", streamID.String()).
		Int64("bytes_sent", stream.BytesSent).
		Int64("bytes_recv", stream.BytesRecv).
		Msg("Stream closed")
}

// Close closes the client connection
func (tc *TunnelClient) Close() error {
	tc.closeMutex.Lock()
	if tc.closed {
		tc.closeMutex.Unlock()
		return nil
	}
	tc.closed = true
	tc.closeMutex.Unlock()

	// Close done channel
	select {
	case <-tc.done:
	default:
		close(tc.done)
	}

	// Close all streams
	tc.streamMux.Lock()
	for _, stream := range tc.streams {
		select {
		case <-stream.Done:
		default:
			close(stream.Done)
		}
		stream.LocalConn.Close()
	}
	tc.streams = make(map[protocol.StreamID]*LocalStream)
	tc.streamMux.Unlock()

	// Close WebSocket connection
	if tc.conn != nil {
		tc.conn.Close()
	}

	tc.logger.Info().Msg("Client closed")
	return nil
}

// GetServerInfo returns the server information
func (tc *TunnelClient) GetServerInfo() *protocol.ServerHello {
	return tc.serverInfo
}

// RotateToNextServer rotates to the next server in the cluster
func (tc *TunnelClient) RotateToNextServer() {
	tc.currentServerIdx = (tc.currentServerIdx + 1) % len(tc.serverList)
	tc.logger.Info().
		Int("new_server_index", tc.currentServerIdx).
		Int("total_servers", len(tc.serverList)).
		Str("server", fmt.Sprintf("%s:%d", tc.serverList[tc.currentServerIdx].Host, tc.serverList[tc.currentServerIdx].Port)).
		Msg("Rotated to next server")
}

// GetCurrentServer returns the current server info
func (tc *TunnelClient) GetCurrentServer() config.ServerNode {
	return tc.serverList[tc.currentServerIdx]
}

// GetServerCount returns the number of servers in the cluster
func (tc *TunnelClient) GetServerCount() int {
	return len(tc.serverList)
}

// GetActiveStreams returns the number of active streams
func (tc *TunnelClient) GetActiveStreams() int {
	tc.streamMux.RLock()
	defer tc.streamMux.RUnlock()
	return len(tc.streams)
}
