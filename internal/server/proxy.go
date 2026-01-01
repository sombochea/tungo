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
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to initialize stream")
	}

	if err := client.SendMessage(msg); err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("Failed to send init message")
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
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to build request")
	}

	// Send request data
	dataMsg := &protocol.DataMessage{
		Data: requestData,
	}
	msg, err = protocol.NewMessage(protocol.MessageTypeData, streamID, dataMsg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create data message")
	}

	if err := client.SendMessage(msg); err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("Failed to send request data")
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
				return ph.sendHTTPResponse(c, responseBuffer)
			}
			return c.Status(fiber.StatusBadGateway).SendString("No response data received")

		case <-stream.Done:
			if responseBuffer.Len() > 0 {
				return ph.sendHTTPResponse(c, responseBuffer)
			}
			return c.Status(fiber.StatusBadGateway).SendString("Stream closed without response")

		case <-timeout:
			return c.Status(fiber.StatusGatewayTimeout).SendString("Request timeout")
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
func (ph *ProxyHandler) sendHTTPResponse(c fiber.Ctx, responseBuffer *bytes.Buffer) error {
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
		return c.Status(fiber.StatusOK).Send(data)
	}

	// Check if response starts with HTTP
	if !bytes.HasPrefix(data, []byte("HTTP/")) {
		ph.logger.Warn().
			Str("start", string(data[:min(20, len(data))])).
			Msg("Response doesn't start with HTTP/, returning as-is")
		// Return as plain text instead of error
		c.Set("Content-Type", "text/plain")
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
		return c.Status(fiber.StatusBadGateway).SendString("Invalid response from backend")
	}
	defer resp.Body.Close()

	// Set status code
	c.Status(resp.StatusCode)

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
		return c.Status(fiber.StatusBadGateway).SendString("Failed to read backend response")
	}

	return c.Send(body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
