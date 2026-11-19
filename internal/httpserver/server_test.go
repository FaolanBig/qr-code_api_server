package httpserver

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testHandler() http.Handler {
	return New("test-api-key-123456", slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

func makeServerTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{R: 30, G: 74, B: 200, A: 255})
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
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

func TestLogoBase64(t *testing.T) {
	logo := makeServerTestPNG(t)
	body := `{"url":"https://example.com","show_logo":true,"logo_png_base64":"` + base64.StdEncoding.EncodeToString(logo) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/url", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-api-key-123456")
	response := httptest.NewRecorder()
	New("test-api-key-123456", nil, WithFallbackLogo(makeServerTestPNG(t)), WithLogoMaxBytes(1024*1024)).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"logo_included":true`) || !strings.Contains(response.Body.String(), `"logo_source":"custom"`) {
		t.Fatalf("missing logo metadata: %s", response.Body.String())
	}
}

func TestFallbackLogo(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/text", strings.NewReader(`{"text":"Hello","show_logo":true}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-api-key-123456")
	response := httptest.NewRecorder()
	New("test-api-key-123456", nil, WithFallbackLogo(makeServerTestPNG(t))).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"logo_source":"fallback"`) {
		t.Fatalf("missing fallback logo metadata: %s", response.Body.String())
	}
}

func TestInvalidLogoBase64(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/url", strings.NewReader(`{"url":"https://example.com","show_logo":true,"logo_png_base64":"bm90LXBuZw=="}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-api-key-123456")
	response := httptest.NewRecorder()
	New("test-api-key-123456", nil).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("got status %d", response.Code)
	}
}

func TestMultipartLogo(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("url", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("show_logo", "true"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(makeServerTestPNG(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/qr/url", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-API-Key", "test-api-key-123456")
	response := httptest.NewRecorder()
	New("test-api-key-123456", nil).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"logo_source":"custom"`) {
		t.Fatalf("missing multipart logo metadata: %s", response.Body.String())
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
