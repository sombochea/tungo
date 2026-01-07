package protocol

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ClientID represents a unique client identifier
type ClientID string

// GenerateClientID creates a new random client ID
func GenerateClientID() ClientID {
	return ClientID(uuid.New().String())
}

// String returns the string representation of the client ID
func (c ClientID) String() string {
	return string(c)
}

// SecretKey represents an API authentication key
type SecretKey struct {
	Key string `json:"key"`
}

// GenerateSecretKey creates a new random secret key
func GenerateSecretKey() (*SecretKey, error) {
	b := make([]byte, 22)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}
	return &SecretKey{
		Key: base64.URLEncoding.EncodeToString(b),
	}, nil
}

// ClientIDFromKey derives a client ID from the secret key using SHA256
func (s *SecretKey) ClientIDFromKey() ClientID {
	hash := sha256.Sum256([]byte(s.Key))
	return ClientID(base64.StdEncoding.EncodeToString(hash[:]))
}

// ReconnectToken represents a token for reconnecting to an existing tunnel
type ReconnectToken struct {
	Token string `json:"token"`
}

// GenerateReconnectToken creates a new reconnect token
func GenerateReconnectToken() (*ReconnectToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate reconnect token: %w", err)
	}
	return &ReconnectToken{
		Token: base64.URLEncoding.EncodeToString(b),
	}, nil
}

// ClientType represents the type of client connection
type ClientType string

const (
	ClientTypeAuth      ClientType = "auth"
	ClientTypeAnonymous ClientType = "anonymous"
)

// ClientHello represents the initial message from client to server
type ClientHello struct {
	ID             ClientID        `json:"id"`
	SubDomain      *string         `json:"sub_domain,omitempty"`
	ClientType     ClientType      `json:"client_type"`
	ClientVersion  string          `json:"client_version,omitempty"`
	SecretKey      *SecretKey      `json:"secret_key,omitempty"`
	ReconnectToken *ReconnectToken `json:"reconnect_token,omitempty"`
	Password       *string         `json:"password,omitempty"` // Optional password to protect tunnel access
}

// NewClientHello creates a new client hello message
func NewClientHello(subDomain *string, secretKey *SecretKey) *ClientHello {
	hello := &ClientHello{
		ID:        GenerateClientID(),
		SubDomain: subDomain,
	}

	if secretKey != nil {
		hello.ClientType = ClientTypeAuth
		hello.SecretKey = secretKey
	} else {
		hello.ClientType = ClientTypeAnonymous
	}

	return hello
}

// SetClientVersion sets the client version for the hello message
func (h *ClientHello) SetClientVersion(version string) {
	h.ClientVersion = version
}

// NewReconnectHello creates a client hello message for reconnection
func NewReconnectHello(token *ReconnectToken) *ClientHello {
	return &ClientHello{
		ID:             GenerateClientID(),
		ClientType:     ClientTypeAnonymous,
		ReconnectToken: token,
	}
}

// ServerHelloType represents the type of server hello response
type ServerHelloType string

const (
	ServerHelloSuccess          ServerHelloType = "success"
	ServerHelloSubDomainInUse   ServerHelloType = "sub_domain_in_use"
	ServerHelloInvalidSubDomain ServerHelloType = "invalid_sub_domain"
	ServerHelloAuthFailed       ServerHelloType = "auth_failed"
	ServerHelloError            ServerHelloType = "error"
)

// ServerHello represents the server's response to a client hello
type ServerHello struct {
	Type           ServerHelloType `json:"type"`
	SubDomain      string          `json:"sub_domain,omitempty"`
	Hostname       string          `json:"hostname,omitempty"`
	PublicURL      string          `json:"public_url,omitempty"`
	ClientID       ClientID        `json:"client_id,omitempty"`
	ReconnectToken *ReconnectToken `json:"reconnect_token,omitempty"`
	Error          string          `json:"error,omitempty"`
}

// NewSuccessHello creates a success server hello
func NewSuccessHello(subDomain, hostname, publicURL string, clientID ClientID, token *ReconnectToken) *ServerHello {
	return &ServerHello{
		Type:           ServerHelloSuccess,
		SubDomain:      subDomain,
		Hostname:       hostname,
		PublicURL:      publicURL,
		ClientID:       clientID,
		ReconnectToken: token,
	}
}

// NewErrorHello creates an error server hello
func NewErrorHello(helloType ServerHelloType, errorMsg string) *ServerHello {
	return &ServerHello{
		Type:  helloType,
		Error: errorMsg,
	}
}

// StreamID represents a unique stream identifier
type StreamID string

// GenerateStreamID creates a new random stream ID
func GenerateStreamID() StreamID {
	return StreamID(uuid.New().String())
}

// String returns the string representation of the stream ID
func (s StreamID) String() string {
	return string(s)
}

// MessageType represents the type of message being sent
type MessageType string

const (
	MessageTypeHello       MessageType = "hello"
	MessageTypeServerHello MessageType = "server_hello"
	MessageTypeInit        MessageType = "init"
	MessageTypeData        MessageType = "data"
	MessageTypeEnd         MessageType = "end"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
)

// Message represents a message in the tunnel protocol
type Message struct {
	Type     MessageType     `json:"type"`
	StreamID StreamID        `json:"stream_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// NewMessage creates a new protocol message
func NewMessage(msgType MessageType, streamID StreamID, data interface{}) (*Message, error) {
	msg := &Message{
		Type:     msgType,
		StreamID: streamID,
	}

	if data != nil {
		dataBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message data: %w", err)
		}
		msg.Data = dataBytes
	}

	return msg, nil
}

// Unmarshal unmarshals the message data into the provided interface
func (m *Message) Unmarshal(v interface{}) error {
	if m.Data == nil {
		return fmt.Errorf("message has no data")
	}
	return json.Unmarshal(m.Data, v)
}

// InitStreamMessage represents a message to initialize a new stream
type InitStreamMessage struct {
	StreamID StreamID `json:"stream_id"`
	Protocol string   `json:"protocol"` // "http", "https", etc.
}

// DataMessage represents a message containing stream data
type DataMessage struct {
	Data []byte `json:"data"`
}

// ValidateSubDomain checks if a subdomain is valid
func ValidateSubDomain(subDomain string) error {
	if len(subDomain) == 0 {
		return fmt.Errorf("subdomain cannot be empty")
	}

	if len(subDomain) > 63 {
		return fmt.Errorf("subdomain too long (max 63 characters)")
	}

	// Check for valid characters (alphanumeric and hyphens)
	for i, c := range subDomain {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("subdomain contains invalid character: %c", c)
		}
		// Cannot start or end with hyphen
		if c == '-' && (i == 0 || i == len(subDomain)-1) {
			return fmt.Errorf("subdomain cannot start or end with hyphen")
		}
	}

	return nil
}

// GenerateRandomSubDomain generates a random subdomain
func GenerateRandomSubDomain() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random subdomain: %w", err)
	}
	// Encode to base64 and make it URL-safe and lowercase
	encoded := base64.URLEncoding.EncodeToString(b)
	encoded = strings.ToLower(encoded)
	encoded = strings.ReplaceAll(encoded, "_", "")
	encoded = strings.ReplaceAll(encoded, "-", "")
	if len(encoded) > 8 {
		encoded = encoded[:8]
	}
	return encoded, nil
}

// EncodeMessage encodes a message to JSON bytes
func EncodeMessage(msg *Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return data, nil
}

// DecodeMessage decodes a message from JSON bytes
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}
	return &msg, nil
}
