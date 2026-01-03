package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"github.com/sombochea/tungo/pkg/protocol"
)

// Buffer pool for high performance
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// ProxyHandler handles HTTP requests and routes them through tunnels
type ProxyHandler struct {
	connMgr *ConnectionManager
	logger  zerolog.Logger
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(connMgr *ConnectionManager, logger zerolog.Logger) *ProxyHandler {
	return &ProxyHandler{
		connMgr: connMgr,
		logger:  logger,
	}
}

// HandleRequest handles an incoming HTTP request
func (ph *ProxyHandler) HandleRequest(c fiber.Ctx, client *ClientConnection) error {
	// Generate stream ID
	streamID := protocol.GenerateStreamID()

	ph.logger.Debug().
		Str("stream_id", streamID.String()).
		Str("client_id", client.ID.String()).
		Str("subdomain", client.SubDomain).
		Str("path", c.Path()).
		Str("method", c.Method()).
		Msg("Handling request")

	// Add stream to client
	stream := client.AddStream(streamID, "http", c.IP())
	defer client.RemoveStream(streamID)

	// Send init message to client
	initMsg := &protocol.InitStreamMessage{
		StreamID: streamID,
		Protocol: "http",
	}

	msg, err := protocol.NewMessage(protocol.MessageTypeInit, streamID, initMsg)
	if err != nil {
		return ph.sendPrettyError(c, fiber.StatusInternalServerError,
			"Stream Initialization Failed",
			"Unable to initialize the tunnel stream. Please try reconnecting your tunnel client.")
	}

	if err := client.SendMessage(msg); err != nil {
		return ph.sendPrettyError(c, fiber.StatusBadGateway,
			"Communication Error",
			"Failed to communicate with the tunnel client. The connection may be unstable.")
	}

	// Wait for the stream to be added to the client (with timeout)
	streamReady := false
	for i := 0; i < 50; i++ {
		if _, exists := client.GetStream(streamID); exists {
			streamReady = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !streamReady {
		ph.logger.Warn().Str("stream_id", streamID.String()).Msg("Stream not ready after init")
	}

	// Build HTTP request data
	requestData, err := ph.buildHTTPRequest(c)
	if err != nil {
		return ph.sendPrettyError(c, fiber.StatusInternalServerError,
			"Request Processing Error",
			"Unable to process your request. Please check your request format and try again.")
	}

	// Send request data
	dataMsg := &protocol.DataMessage{
		Data: requestData,
	}
	msg, err = protocol.NewMessage(protocol.MessageTypeData, streamID, dataMsg)
	if err != nil {
		return ph.sendPrettyError(c, fiber.StatusInternalServerError,
			"Message Creation Failed",
			"Unable to create tunnel message. Please try again.")
	}

	if err := client.SendMessage(msg); err != nil {
		return ph.sendPrettyError(c, fiber.StatusBadGateway,
			"Data Transmission Failed",
			"Unable to send your request through the tunnel. The connection may have been interrupted.")
	}

	// Wait for response data with timeout
	timeout := time.After(30 * time.Second)
	responseBuffer := bufferPool.Get().(*bytes.Buffer)
	responseBuffer.Reset()
	defer bufferPool.Put(responseBuffer)

	noDataTimeout := time.NewTimer(5 * time.Second) // Initial timeout for first response
	defer noDataTimeout.Stop()

	for {
		select {
		case data := <-stream.DataChan:
			ph.logger.Info().
				Str("stream_id", streamID.String()).
				Int("chunk_bytes", len(data)).
				Int("total_bytes", responseBuffer.Len()).
				Str("chunk_preview", string(data[:min(50, len(data))])).
				Msg("Received response chunk")

			responseBuffer.Write(data)
			// Reset the no-data timeout since we received data
			// Use shorter timeout for subsequent data chunks
			noDataTimeout.Reset(200 * time.Millisecond)

		case <-noDataTimeout.C:
			// No more data coming, parse and return HTTP response
			if responseBuffer.Len() > 0 {
				return ph.sendHTTPResponse(c, responseBuffer, client, streamID, stream)
			}
			return ph.sendPrettyErrorWithInfo(c, fiber.StatusBadGateway,
				"No Response Received",
				"Your local server didn't respond. Please check if your local application is running and accessible.",
				client, streamID, stream)

		case <-stream.Done:
			if responseBuffer.Len() > 0 {
				return ph.sendHTTPResponse(c, responseBuffer, client, streamID, stream)
			}
			return ph.sendPrettyErrorWithInfo(c, fiber.StatusBadGateway,
				"Connection Closed",
				"The tunnel connection was closed before receiving a response. Your local server may have stopped or crashed.",
				client, streamID, stream)

		case <-timeout:
			return ph.sendPrettyErrorWithInfo(c, fiber.StatusGatewayTimeout,
				"Request Timeout",
				"Your local server took too long to respond (>30s). Please check if your application is experiencing performance issues.",
				client, streamID, stream)
		}
	}
}

// buildHTTPRequest builds an HTTP request from Fiber context
func (ph *ProxyHandler) buildHTTPRequest(c fiber.Ctx) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// Request line
	method := c.Method()
	path := c.Path()
	if query := string(c.Request().URI().QueryString()); query != "" {
		path += "?" + query
	}
	fmt.Fprintf(buf, "%s %s HTTP/1.1\r\n", method, path)

	// Headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		fmt.Fprintf(buf, "%s: %s\r\n", key, value)
	})

	// Host header
	if c.Request().Header.Peek("Host") == nil {
		fmt.Fprintf(buf, "Host: localhost\r\n")
	}

	// End of headers
	fmt.Fprintf(buf, "\r\n")

	// Body
	if len(c.Body()) > 0 {
		buf.Write(c.Body())
	}

	return buf.Bytes(), nil
}

// sendHTTPResponse parses raw HTTP response and sends it through Fiber
func (ph *ProxyHandler) sendHTTPResponse(c fiber.Ctx, responseBuffer *bytes.Buffer, client *ClientConnection, streamID protocol.StreamID, stream *Stream) error {
	data := responseBuffer.Bytes()

	// Log first 200 bytes for debugging
	previewLen := 200
	if len(data) < previewLen {
		previewLen = len(data)
	}
	ph.logger.Info().
		Int("total_bytes", len(data)).
		Str("preview", string(data[:previewLen])).
		Msg("Parsing HTTP response")

	// Validate we have at least some data that looks like HTTP
	if len(data) < 12 { // Minimum: "HTTP/1.0 200"
		ph.logger.Warn().Int("bytes", len(data)).Msg("Response too short to be valid HTTP, returning as-is")
		// Return as plain text instead of error
		c.Set("Content-Type", "text/plain")
		// Add TunGo headers even for non-HTTP responses
		setTunGoHeaders(c, client, streamID, stream)
		return c.Status(fiber.StatusOK).Send(data)
	}

	// Check if response starts with HTTP
	if !bytes.HasPrefix(data, []byte("HTTP/")) {
		ph.logger.Warn().
			Str("start", string(data[:min(20, len(data))])).
			Msg("Response doesn't start with HTTP/, returning as-is")
		// Return as plain text instead of error
		c.Set("Content-Type", "text/plain")
		// Add TunGo headers even for non-HTTP responses
		setTunGoHeaders(c, client, streamID, stream)
		return c.Status(fiber.StatusOK).Send(data)
	}

	// Parse HTTP response
	reader := bufio.NewReader(responseBuffer)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		ph.logger.Error().
			Err(err).
			Int("buffer_size", len(data)).
			Str("buffer_preview", string(data[:min(100, len(data))])).
			Msg("Failed to parse HTTP response")
		return ph.sendPrettyError(c, fiber.StatusBadGateway,
			"Invalid Response",
			"Your local server returned an invalid HTTP response. Please ensure your application is sending properly formatted HTTP responses.")
	}
	defer resp.Body.Close()

	// Set status code
	c.Status(resp.StatusCode)

	// Add TunGo custom headers for tunnel information
	setTunGoHeaders(c, client, streamID, stream)

	// Copy headers (preserve all headers including Content-Type)
	for key, values := range resp.Header {
		for _, value := range values {
			c.Set(key, value)
		}
	}

	// Read and send body efficiently
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ph.logger.Error().Err(err).Msg("Failed to read response body")
		return ph.sendPrettyError(c, fiber.StatusBadGateway,
			"Response Read Error",
			"Unable to read the full response from your local server. The connection may have been interrupted.")
	}

	return c.Send(body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// setTunGoHeaders adds TunGo custom headers to the response
func setTunGoHeaders(c fiber.Ctx, client *ClientConnection, streamID protocol.StreamID, stream *Stream) {
	protocolType := "unknown"
	if stream != nil {
		protocolType = stream.Protocol
	}

	clientVersion := client.ClientVersion
	if clientVersion == "" {
		clientVersion = "unknown"
	}

	c.Set("X-Tungo-Client-ID", client.ID.String())
	c.Set("X-Tungo-Stream-ID", streamID.String())
	c.Set("X-Tungo-Subdomain", client.SubDomain)
	c.Set("X-Tungo-Protocol", protocolType)
	c.Set("X-Tungo-Version", clientVersion)
}

// sendPrettyError sends a user-friendly HTML error response
func (ph *ProxyHandler) sendPrettyError(c fiber.Ctx, status int, title, message string) error {
	return ph.sendPrettyErrorWithInfo(c, status, title, message, nil, "", nil)
}

// sendPrettyErrorWithInfo sends a user-friendly HTML error response with optional tunnel info
func (ph *ProxyHandler) sendPrettyErrorWithInfo(c fiber.Ctx, status int, title, message string, client *ClientConnection, streamID protocol.StreamID, stream *Stream) error {
	// Add TunGo headers if client info is provided
	if client != nil {
		setTunGoHeaders(c, client, streamID, stream)
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }
        .error-container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 60px 40px;
            max-width: 600px;
            text-align: center;
        }
        .error-icon {
            font-size: 72px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            font-size: 32px;
            margin-bottom: 16px;
            font-weight: 700;
        }
        p {
            color: #666;
            font-size: 18px;
            line-height: 1.6;
            margin-bottom: 32px;
        }
        .status-code {
            display: inline-block;
            background: #f0f0f0;
            color: #888;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 600;
            margin-top: 20px;
        }
        .footer {
            margin-top: 40px;
            color: #999;
            font-size: 14px;
        }
        a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
        }
        a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-icon">🔌</div>
        <h1>%s</h1>
        <p>%s</p>
        <div class="status-code">Status Code: %d</div>
        <div class="footer">
            Powered by <a href="https://github.com/sombochea/tungo">TunGo</a>
        </div>
    </div>
</body>
</html>`, title, title, message, status)
	return c.Status(status).SendString(html)
}

// ParseHTTPResponse is deprecated - use sendHTTPResponse instead
func ParseHTTPResponse(data []byte) (int, []byte, error) {
	// This is a simplified version
	// In production, you'd want to use net/http's response parser

	// Find the end of headers
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return fiber.StatusOK, data, nil
	}

	// Extract body
	body := data[headerEnd+4:]

	return fiber.StatusOK, body, nil
}

// Simple implementation that just forwards raw bytes
func (ph *ProxyHandler) HandleRequestSimple(c fiber.Ctx, client *ClientConnection) error {
	ph.logger.Info().
		Str("method", c.Method()).
		Str("path", c.Path()).
		Str("subdomain", client.SubDomain).
		Msg("Proxying request")

	// For now, return a simple response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Tunnel is active",
		"subdomain": client.SubDomain,
		"method":    c.Method(),
		"path":      c.Path(),
		"status":    "This is a simplified implementation. Full HTTP proxying will forward to your local server.",
	})
}
