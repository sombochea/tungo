package server

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/sombochea/tungo/internal/registry"
	"github.com/sombochea/tungo/pkg/protocol"
)

// ClientConnection represents a connected client
type ClientConnection struct {
	ID            protocol.ClientID
	SubDomain     string
	ClientVersion string
	Conn          *websocket.Conn
	Streams       map[protocol.StreamID]*Stream
	StreamMutex   sync.RWMutex
	Logger        zerolog.Logger
	Send          chan []byte
	Done          chan struct{}
}

// Stream represents an active data stream
type Stream struct {
	ID         protocol.StreamID
	Protocol   string
	RemoteAddr string
	DataChan   chan []byte
	Done       chan struct{}
}

// ConnectionManager manages all active client connections
type ConnectionManager struct {
	clients       map[protocol.ClientID]*ClientConnection
	subdomains    map[string]protocol.ClientID
	mutex         sync.RWMutex
	registry      registry.Registry
	logger        zerolog.Logger
	maxConnection int
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(reg registry.Registry, logger zerolog.Logger, maxConn int) *ConnectionManager {
	return &ConnectionManager{
		clients:       make(map[protocol.ClientID]*ClientConnection),
		subdomains:    make(map[string]protocol.ClientID),
		registry:      reg,
		logger:        logger,
		maxConnection: maxConn,
	}
}

// AddClient adds a new client connection
func (cm *ConnectionManager) AddClient(clientID protocol.ClientID, subDomain string, clientVersion string, conn *websocket.Conn) (*ClientConnection, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if max connections reached
	if len(cm.clients) >= cm.maxConnection {
		return nil, fmt.Errorf("maximum connections reached")
	}

	// Check if subdomain is already in use
	if existingID, exists := cm.subdomains[subDomain]; exists {
		if existingID != clientID {
			return nil, fmt.Errorf("subdomain already in use")
		}
	}

	client := &ClientConnection{
		ID:            clientID,
		SubDomain:     subDomain,
		ClientVersion: clientVersion,
		Conn:          conn,
		Streams:       make(map[protocol.StreamID]*Stream),
		Logger:        cm.logger.With().Str("client_id", clientID.String()).Str("subdomain", subDomain).Logger(),
		Send:          make(chan []byte, 512), // Increased buffer for high throughput
		Done:          make(chan struct{}),
	}

	cm.clients[clientID] = client
	cm.subdomains[subDomain] = clientID

	cm.logger.Info().
		Str("client_id", clientID.String()).
		Str("subdomain", subDomain).
		Msg("Client connected")

	return client, nil
}

// RemoveClient removes a client connection
func (cm *ConnectionManager) RemoveClient(clientID protocol.ClientID) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	client, exists := cm.clients[clientID]
	if !exists {
		return
	}

	// Clean up subdomain mapping
	delete(cm.subdomains, client.SubDomain)

	// Close all streams
	client.StreamMutex.Lock()
	for _, stream := range client.Streams {
		close(stream.Done)
	}
	client.Streams = make(map[protocol.StreamID]*Stream)
	client.StreamMutex.Unlock()

	// Close done channel
	close(client.Done)

	// Remove client
	delete(cm.clients, clientID)

	cm.logger.Info().
		Str("client_id", clientID.String()).
		Str("subdomain", client.SubDomain).
		Msg("Client disconnected")
}

// GetClient retrieves a client by ID
func (cm *ConnectionManager) GetClient(clientID protocol.ClientID) (*ClientConnection, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	client, exists := cm.clients[clientID]
	return client, exists
}

// GetClientBySubDomain retrieves a client by subdomain
func (cm *ConnectionManager) GetClientBySubDomain(subDomain string) (*ClientConnection, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	clientID, exists := cm.subdomains[subDomain]
	if !exists {
		return nil, false
	}

	client, exists := cm.clients[clientID]
	return client, exists
}

// IsSubDomainAvailable checks if a subdomain is available
func (cm *ConnectionManager) IsSubDomainAvailable(subDomain string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	_, exists := cm.subdomains[subDomain]
	return !exists
}

// GetActiveConnections returns the number of active connections
func (cm *ConnectionManager) GetActiveConnections() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.clients)
}

// ListSubDomains returns all active subdomains
func (cm *ConnectionManager) ListSubDomains() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	subdomains := make([]string, 0, len(cm.subdomains))
	for subdomain := range cm.subdomains {
		subdomains = append(subdomains, subdomain)
	}
	return subdomains
}

// AddStream adds a new stream to a client
func (cc *ClientConnection) AddStream(streamID protocol.StreamID, protocol, remoteAddr string) *Stream {
	cc.StreamMutex.Lock()
	defer cc.StreamMutex.Unlock()

	stream := &Stream{
		ID:         streamID,
		Protocol:   protocol,
		RemoteAddr: remoteAddr,
		DataChan:   make(chan []byte, 512), // Increased buffer for high throughput
		Done:       make(chan struct{}),
	}

	cc.Streams[streamID] = stream

	cc.Logger.Debug().
		Str("stream_id", streamID.String()).
		Str("protocol", protocol).
		Str("remote_addr", remoteAddr).
		Msg("Stream added")

	return stream
}

// GetStream retrieves a stream by ID
func (cc *ClientConnection) GetStream(streamID protocol.StreamID) (*Stream, bool) {
	cc.StreamMutex.RLock()
	defer cc.StreamMutex.RUnlock()
	stream, exists := cc.Streams[streamID]
	return stream, exists
}

// RemoveStream removes a stream from a client
func (cc *ClientConnection) RemoveStream(streamID protocol.StreamID) {
	cc.StreamMutex.Lock()
	defer cc.StreamMutex.Unlock()

	stream, exists := cc.Streams[streamID]
	if !exists {
		return
	}

	close(stream.Done)
	delete(cc.Streams, streamID)

	cc.Logger.Debug().
		Str("stream_id", streamID.String()).
		Msg("Stream removed")
}

// GetActiveStreams returns the number of active streams
func (cc *ClientConnection) GetActiveStreams() int {
	cc.StreamMutex.RLock()
	defer cc.StreamMutex.RUnlock()
	return len(cc.Streams)
}

// SendMessage sends a message to the client
func (cc *ClientConnection) SendMessage(msg *protocol.Message) error {
	data, err := protocol.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	select {
	case cc.Send <- data:
		return nil
	case <-cc.Done:
		return fmt.Errorf("client connection closed")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// GetActiveConnectionsCount returns the total number of active client connections
func (cm *ConnectionManager) GetActiveConnectionsCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.clients)
}
