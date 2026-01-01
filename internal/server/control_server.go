package server

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/sombochea/tungo/internal/registry"
	"github.com/sombochea/tungo/pkg/config"
	"github.com/sombochea/tungo/pkg/protocol"
)

// ControlServer handles client control connections
type ControlServer struct {
	config       *config.ServerConfig
	connMgr      *ConnectionManager
	logger       zerolog.Logger
	distRegistry *registry.DistributedRegistry
}

// NewControlServer creates a new control server
func NewControlServer(
	cfg *config.ServerConfig,
	connMgr *ConnectionManager,
	logger zerolog.Logger,
	distRegistry *registry.DistributedRegistry,
) *ControlServer {
	return &ControlServer{
		config:       cfg,
		connMgr:      connMgr,
		logger:       logger,
		distRegistry: distRegistry,
	}
}

// HandleConnection handles a new WebSocket connection
func (cs *ControlServer) HandleConnection(c *websocket.Conn) {
	defer c.Close()

	logger := cs.logger.With().Str("remote_addr", c.RemoteAddr().String()).Logger()
	logger.Info().Msg("New WebSocket connection")

	// Read initial client hello
	var clientHello protocol.ClientHello
	if err := c.ReadJSON(&clientHello); err != nil {
		logger.Error().Err(err).Msg("Failed to read client hello")
		cs.sendErrorHello(c, protocol.ServerHelloError, "Failed to read client hello")
		return
	}

	logger = logger.With().Str("client_id", clientHello.ID.String()).Logger()

	// Handle authentication
	serverHello, clientID, subDomain, err := cs.authenticate(&clientHello)
	if err != nil {
		logger.Error().Err(err).Msg("Authentication failed")
		cs.sendServerHello(c, serverHello)
		return
	}

	// Add client to connection manager (fully in-memory, stateless)
	clientConn, err := cs.connMgr.AddClient(clientID, subDomain, c)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to add client")
		cs.sendErrorHello(c, protocol.ServerHelloError, err.Error())
		return
	}
	defer func() {
		cs.connMgr.RemoveClient(clientID)
		// Unregister from distributed registry if enabled
		if cs.distRegistry != nil {
			if err := cs.distRegistry.UnregisterTunnel(subDomain); err != nil {
				logger.Error().Err(err).Msg("Failed to unregister tunnel from registry")
			}
		}
	}()

	logger.Info().
		Str("client_id", clientID.String()).
		Str("subdomain", subDomain).
		Bool("distributed", cs.distRegistry != nil).
		Msg("Client connected")

	// Register tunnel in distributed registry if enabled
	if cs.distRegistry != nil {
		tunnelInfo := &registry.TunnelInfo{
			Subdomain:   subDomain,
			ServerHost:  cs.config.Host,
			ClientID:    clientID.String(),
			ProxyPort:   cs.config.Port,
			ControlPort: cs.config.ControlPort,
			CreatedAt:   time.Now(),
		}
		if err := cs.distRegistry.RegisterTunnel(tunnelInfo); err != nil {
			logger.Error().Err(err).Msg("Failed to register tunnel in distributed registry")
			// Don't fail the connection, continue anyway
		} else {
			logger.Info().Str("subdomain", subDomain).Msg("Tunnel registered in distributed registry")
		}
	}

	// Send success response
	if err := cs.sendServerHello(c, serverHello); err != nil {
		logger.Error().Err(err).Msg("Failed to send server hello")
		return
	}

	logger.Info().
		Str("subdomain", subDomain).
		Str("hostname", serverHello.Hostname).
		Msg("Client authenticated and tunnel established")

	// Start goroutines for reading and writing
	go cs.writePump(clientConn)
	cs.readPump(clientConn)
}

// authenticate authenticates a client hello message (stateless)
func (cs *ControlServer) authenticate(hello *protocol.ClientHello) (*protocol.ServerHello, protocol.ClientID, string, error) {
	var clientID protocol.ClientID
	var subDomain string

	// Handle authentication (stateless)
	if hello.ClientType == protocol.ClientTypeAuth {
		if hello.SecretKey == nil {
			return protocol.NewErrorHello(protocol.ServerHelloAuthFailed, "Secret key required"), "", "", fmt.Errorf("secret key required")
		}

		// Derive client ID from secret key (deterministic)
		clientID = hello.SecretKey.ClientIDFromKey()

		// Check if subdomain is specified or client is reconnecting
		if hello.SubDomain != nil {
			if err := protocol.ValidateSubDomain(*hello.SubDomain); err != nil {
				return protocol.NewErrorHello(protocol.ServerHelloInvalidSubDomain, err.Error()), "", "", err
			}
			subDomain = *hello.SubDomain
		} else {
			randomSub, err := protocol.GenerateRandomSubDomain()
			if err != nil {
				return protocol.NewErrorHello(protocol.ServerHelloError, "Failed to generate subdomain"), "", "", err
			}
			subDomain = randomSub
		}

		// Check if subdomain is available (in-memory only)
		if !cs.connMgr.IsSubDomainAvailable(subDomain) {
			return protocol.NewErrorHello(protocol.ServerHelloSubDomainInUse, "Subdomain is already in use"), "", "", fmt.Errorf("subdomain in use")
		}
	} else {
		// Anonymous client
		if !cs.config.AllowAnonymous {
			return protocol.NewErrorHello(protocol.ServerHelloAuthFailed, "Anonymous clients not allowed"), "", "", fmt.Errorf("anonymous not allowed")
		}

		clientID = hello.ID

		// Generate subdomain
		if hello.SubDomain != nil {
			if err := protocol.ValidateSubDomain(*hello.SubDomain); err != nil {
				return protocol.NewErrorHello(protocol.ServerHelloInvalidSubDomain, err.Error()), "", "", err
			}
			subDomain = *hello.SubDomain
		} else {
			randomSub, err := protocol.GenerateRandomSubDomain()
			if err != nil {
				return protocol.NewErrorHello(protocol.ServerHelloError, "Failed to generate subdomain"), "", "", err
			}
			subDomain = randomSub
		}

		// Check if subdomain is available (in-memory only)
		if !cs.connMgr.IsSubDomainAvailable(subDomain) {
			return protocol.NewErrorHello(protocol.ServerHelloSubDomainInUse, "Subdomain is already in use"), "", "", fmt.Errorf("subdomain in use")
		}
	}

	// Create success response (stateless, no reconnect token needed)
	hostname := fmt.Sprintf("%s.%s", subDomain, cs.config.SubDomainSuffix)
	serverHello := protocol.NewSuccessHello(subDomain, hostname, clientID, nil)

	return serverHello, clientID, subDomain, nil
}

// readPump reads messages from the WebSocket connection
func (cs *ControlServer) readPump(client *ClientConnection) {
	defer func() {
		cs.connMgr.RemoveClient(client.ID)
	}()

	for {
		var msg protocol.Message
		if err := client.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				client.Logger.Error().Err(err).Msg("WebSocket read error")
			}
			break
		}

		cs.handleMessage(client, &msg)
	}
}

// writePump writes messages to the WebSocket connection
func (cs *ControlServer) writePump(client *ClientConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				client.Logger.Error().Err(err).Msg("WebSocket write error")
				return
			}

		case <-ticker.C:
			// Send ping
			pingMsg, _ := protocol.NewMessage(protocol.MessageTypePing, "", nil)
			data, _ := protocol.EncodeMessage(pingMsg)
			if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				client.Logger.Error().Err(err).Msg("Failed to send ping")
				return
			}

		case <-client.Done:
			return
		}
	}
}

// handleMessage handles a received message
func (cs *ControlServer) handleMessage(client *ClientConnection, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypePong:
		client.Logger.Debug().Msg("Received pong")

	case protocol.MessageTypeData:
		stream, exists := client.GetStream(msg.StreamID)
		if !exists {
			client.Logger.Warn().Str("stream_id", msg.StreamID.String()).Msg("Stream not found for data message")
			return
		}

		var dataMsg protocol.DataMessage
		if err := msg.Unmarshal(&dataMsg); err != nil {
			client.Logger.Error().Err(err).Msg("Failed to unmarshal data message")
			return
		}

		// Debug: log received data
		previewLen := 100
		if len(dataMsg.Data) < previewLen {
			previewLen = len(dataMsg.Data)
		}
		client.Logger.Info().
			Str("stream_id", msg.StreamID.String()).
			Int("bytes", len(dataMsg.Data)).
			Str("preview", string(dataMsg.Data[:previewLen])).
			Msg("Received DATA from client")

		select {
		case stream.DataChan <- dataMsg.Data:
		case <-stream.Done:
			client.Logger.Debug().Str("stream_id", msg.StreamID.String()).Msg("Stream closed while sending data")
		default:
			client.Logger.Warn().Str("stream_id", msg.StreamID.String()).Msg("Stream data channel full")
		}

	case protocol.MessageTypeEnd:
		client.Logger.Debug().Str("stream_id", msg.StreamID.String()).Msg("Received stream end")
		client.RemoveStream(msg.StreamID)

	default:
		client.Logger.Warn().Str("type", string(msg.Type)).Msg("Unknown message type")
	}
}

// sendServerHello sends a server hello message
func (cs *ControlServer) sendServerHello(c *websocket.Conn, hello *protocol.ServerHello) error {
	return c.WriteJSON(hello)
}

// sendErrorHello sends an error hello message
func (cs *ControlServer) sendErrorHello(c *websocket.Conn, helloType protocol.ServerHelloType, errorMsg string) {
	hello := protocol.NewErrorHello(helloType, errorMsg)
	_ = c.WriteJSON(hello)
}
