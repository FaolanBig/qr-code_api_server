package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testHandler() http.Handler {
	return New("test-api-key-123456", slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

func TestHealthAndReadinessArePublic(t *testing.T) {
	handler := testHandler()
	for _, path := range []string{"/healthz", "/readyz"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s returned %d", path, response.Code)
		}
	}
}

func TestAPIKeyRequired(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want 401", response.Code)
	}
}

func TestURLEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/url", strings.NewReader(`{"url":"https://example.com","size":256,"recovery":"high"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-api-key-123456")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{"\"format\":\"png\"", "\"png_base64\"", "\"data_uri\"", "data:image/png;base64"} {
		if !strings.Contains(body, expected) {
			t.Errorf("response missing %q", expected)
		}
	}
}

func TestInvalidPayloadAndUnknownField(t *testing.T) {
	for _, body := range []string{`{"url":"ftp://example.com"}`, `{"url":"https://example.com","unknown":true}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/url", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer test-api-key-123456")
		response := httptest.NewRecorder()
		testHandler().ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("got status %d for %s", response.Code, body)
		}
	}
}
