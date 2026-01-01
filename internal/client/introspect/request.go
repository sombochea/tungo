package introspect

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Request represents a captured HTTP request/response pair
type Request struct {
	ID              string
	Status          int
	IsReplay        bool
	Path            string
	Method          string
	Headers         [][2]string
	BodyData        []byte
	ResponseHeaders [][2]string
	ResponseData    []byte
	Started         time.Time
	Completed       time.Time
	EntireRequest   []byte
}

// Elapsed returns the duration of the request as a formatted string
func (r *Request) Elapsed() string {
	duration := r.Completed.Sub(r.Started)
	if duration.Seconds() < 1 {
		return duration.Round(time.Millisecond).String()
	}
	return duration.Round(time.Second).String()
}

// RequestStore holds captured requests in memory
type RequestStore struct {
	mu       sync.RWMutex
	requests map[string]*Request
}

var globalStore = &RequestStore{
	requests: make(map[string]*Request),
}

// GetStore returns the global request store
func GetStore() *RequestStore {
	return globalStore
}

// Add adds a request to the store
func (rs *RequestStore) Add(req *Request) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.requests[req.ID] = req
}

// Get retrieves a request by ID
func (rs *RequestStore) Get(id string) (*Request, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	req, ok := rs.requests[id]
	return req, ok
}

// GetAll returns all requests
func (rs *RequestStore) GetAll() []*Request {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	requests := make([]*Request, 0, len(rs.requests))
	for _, req := range rs.requests {
		requests = append(requests, req)
	}
	return requests
}

// Clear removes all requests
func (rs *RequestStore) Clear() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.requests = make(map[string]*Request)
}

// CaptureStream captures HTTP request and response data from raw bytes
func CaptureStream(requestData, responseData []byte) {
	started := time.Now()

	// Parse request
	reqReader := bufio.NewReader(bytes.NewReader(requestData))
	httpReq, err := http.ReadRequest(reqReader)
	if err != nil {
		return // Silently ignore unparseable requests
	}

	// Read request body
	var reqBody []byte
	if httpReq.Body != nil {
		reqBody, _ = io.ReadAll(httpReq.Body)
		httpReq.Body.Close()
	}

	// Convert headers to slice of pairs
	reqHeaders := make([][2]string, 0)
	for name, values := range httpReq.Header {
		for _, value := range values {
			reqHeaders = append(reqHeaders, [2]string{name, value})
		}
	}

	// Parse response
	var status int
	var respHeaders [][2]string
	var respBody []byte

	if len(responseData) > 0 {
		respReader := bufio.NewReader(bytes.NewReader(responseData))
		httpResp, err := http.ReadResponse(respReader, httpReq)
		if err == nil {
			status = httpResp.StatusCode

			// Read response body
			if httpResp.Body != nil {
				respBody, _ = io.ReadAll(httpResp.Body)
				httpResp.Body.Close()
			}

			// Convert headers
			respHeaders = make([][2]string, 0)
			for name, values := range httpResp.Header {
				for _, value := range values {
					respHeaders = append(respHeaders, [2]string{name, value})
				}
			}
		}
	}

	// Create request record
	req := &Request{
		ID:              uuid.New().String(),
		Status:          status,
		IsReplay:        false,
		Path:            httpReq.URL.Path,
		Method:          httpReq.Method,
		Headers:         reqHeaders,
		BodyData:        reqBody,
		ResponseHeaders: respHeaders,
		ResponseData:    respBody,
		Started:         started,
		Completed:       time.Now(),
		EntireRequest:   requestData,
	}

	// Store the request
	GetStore().Add(req)

	// Log to console
	ConsoleLog(httpReq.Method, httpReq.URL.Path, status)
}
