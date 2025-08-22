package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockForwarder implements the Forwarder interface for testing
type MockForwarder struct {
	// ForwardRequestFunc allows us to customize the behavior of ForwardRequest
	ForwardRequestFunc func(w http.ResponseWriter, r *http.Request)
	// CallCount tracks how many times ForwardRequest was called
	CallCount int
	// LastRequest stores the last request that was passed to ForwardRequest
	LastRequest *http.Request
}

func (m *MockForwarder) ForwardRequest(w http.ResponseWriter, r *http.Request) {
	m.CallCount++
	m.LastRequest = r
	
	if m.ForwardRequestFunc != nil {
		m.ForwardRequestFunc(w, r)
	} else {
		// Default behavior: return a simple response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock response"))
	}
}

func TestCUIForwarderHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		setupMock      func(*MockForwarder)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "GET request",
			method: "GET",
			path:   "/test",
			body:   "",
			setupMock: func(m *MockForwarder) {
				m.ForwardRequestFunc = func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("GET response"))
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "GET response",
		},
		{
			name:   "POST request with body",
			method: "POST",
			path:   "/api/data",
			body:   `{"key": "value"}`,
			setupMock: func(m *MockForwarder) {
				m.ForwardRequestFunc = func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					w.Write([]byte(`{"success": true}`))
				}
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"success": true}`,
		},
		{
			name:   "Error response from forwarder",
			method: "GET",
			path:   "/error",
			body:   "",
			setupMock: func(m *MockForwarder) {
				m.ForwardRequestFunc = func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock forwarder
			mockForwarder := &MockForwarder{}
			if tt.setupMock != nil {
				tt.setupMock(mockForwarder)
			}

			// Create handler
			handler := CUIForwarderHandler(mockForwarder)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" && tt.method != "GET" {
				req.Header.Set("Content-Type", "application/json")
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler(w, req)

			// Check that ForwardRequest was called
			if mockForwarder.CallCount != 1 {
				t.Errorf("Expected ForwardRequest to be called once, but was called %d times", mockForwarder.CallCount)
			}

			// Check that the correct request was passed to ForwardRequest
			if mockForwarder.LastRequest == nil {
				t.Fatal("Expected LastRequest to be set")
			}

			if mockForwarder.LastRequest.Method != tt.method {
				t.Errorf("Expected method %s, got %s", tt.method, mockForwarder.LastRequest.Method)
			}

			if mockForwarder.LastRequest.URL.Path != tt.path {
				t.Errorf("Expected path %s, got %s", tt.path, mockForwarder.LastRequest.URL.Path)
			}

			// Check response
			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body := w.Body.String()
			if body != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, body)
			}
		})
	}
}

func TestCUIForwarderHandler_RequestPassthrough(t *testing.T) {
	// Test that the handler properly passes through request details
	mockForwarder := &MockForwarder{}
	
	// Set up mock to capture and verify request details
	mockForwarder.ForwardRequestFunc = func(w http.ResponseWriter, r *http.Request) {
		// Verify headers are passed through
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected custom header to be passed through")
		}
		
		// Verify query parameters are passed through
		if r.URL.RawQuery != "param1=value1&param2=value2" {
			t.Errorf("Expected query parameters to be passed through, got: %s", r.URL.RawQuery)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	handler := CUIForwarderHandler(mockForwarder)

	// Create request with headers and query parameters
	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")

	w := httptest.NewRecorder()
	handler(w, req)

	// Verify the request was properly handled
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCUIForwarderHandler_MultipleHeaders(t *testing.T) {
	// Test that multiple headers with the same name are handled correctly
	mockForwarder := &MockForwarder{}
	
	mockForwarder.ForwardRequestFunc = func(w http.ResponseWriter, r *http.Request) {
		// Check that multiple headers are preserved
		cookies := r.Header["Cookie"]
		if len(cookies) != 2 {
			t.Errorf("Expected 2 Cookie headers, got %d", len(cookies))
		}
		
		w.WriteHeader(http.StatusOK)
	}

	handler := CUIForwarderHandler(mockForwarder)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Add("Cookie", "session=abc123")
	req.Header.Add("Cookie", "theme=dark")

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
