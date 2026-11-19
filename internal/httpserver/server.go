package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	qr "github.com/FaolanBig/qr-code_api_server/internal/qrcode"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const maxRequestBytes = 1 << 20

type Server struct {
	apiKey string
	logger *slog.Logger
}

func New(apiKey string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{apiKey: apiKey, logger: logger}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(securityHeaders)
	r.Use(requestLogger(s.logger))
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.apiKeyMiddleware)
		r.Get("/", s.apiIndex)
		r.Post("/qr/url", s.urlQR)
		r.Post("/qr/vcard", s.vcardQR)
		r.Post("/qr/wifi", s.wifiQR)
		r.Post("/qr/calendar", s.calendarQR)
		r.Post("/qr/text", s.textQR)
		r.Post("/qr/file", s.fileQR)
		r.Post("/qr/number", s.numberQR)
		r.Post("/qr/email", s.emailQR)
		r.Post("/qr/sms", s.smsQR)
		r.Post("/qr/geolocation", s.geolocationQR)
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) apiIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "QR Code API",
		"version": "v1",
		"endpoints": []string{
			"POST /api/v1/qr/url", "POST /api/v1/qr/vcard", "POST /api/v1/qr/wifi",
			"POST /api/v1/qr/calendar", "POST /api/v1/qr/text", "POST /api/v1/qr/file",
			"POST /api/v1/qr/number", "POST /api/v1/qr/email", "POST /api/v1/qr/sms",
			"POST /api/v1/qr/geolocation",
		},
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request: "+err.Error())
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain exactly one JSON object")
		return false
	}
	return true
}

func imageOptions(recovery string, size int) qr.ImageOptions {
	if strings.TrimSpace(recovery) == "" {
		recovery = "medium"
	}
	return qr.ImageOptions{Size: size, Recovery: strings.ToLower(strings.TrimSpace(recovery))}
}

func render(w http.ResponseWriter, payloadType, payload string, options qr.ImageOptions) {
	result, err := qr.Generate(payloadType, payload, options)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"qr_code": result})
}

func (s *Server) urlQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL      string `json:"url"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.URLPayload(body.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "url", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) vcardQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Organization string `json:"organization"`
		Title        string `json:"title"`
		Phone        string `json:"phone"`
		Email        string `json:"email"`
		Website      string `json:"website"`
		Address      string `json:"address"`
		Size         int    `json:"size"`
		Recovery     string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.VCardPayload(body.FirstName, body.LastName, body.Organization, body.Title, body.Phone, body.Email, body.Website, body.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "vcard", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) wifiQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
		Security string `json:"security"`
		Hidden   bool   `json:"hidden"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.WiFiPayload(body.SSID, body.Password, body.Security, body.Hidden)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "wifi", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) calendarQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Location    string `json:"location"`
		Start       string `json:"start"`
		End         string `json:"end"`
		Timezone    string `json:"timezone"`
		AllDay      bool   `json:"all_day"`
		Size        int    `json:"size"`
		Recovery    string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.CalendarPayload(body.Title, body.Description, body.Location, body.Start, body.End, body.Timezone, body.AllDay)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "calendar", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) textQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text     string `json:"text"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.TextPayload(body.Text)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "text", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) fileQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Filename      string `json:"filename"`
		MIMEType      string `json:"mime_type"`
		Base64Content string `json:"base64_content"`
		Size          int    `json:"size"`
		Recovery      string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.FilePayload(body.Filename, body.MIMEType, body.Base64Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "file", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) numberQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Number   string `json:"number"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.NumberPayload(body.Number)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "number", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) emailQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.EmailPayload(body.Email, body.Subject, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "email", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) smsQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone    string `json:"phone"`
		Body     string `json:"body"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.SMSPayload(body.Phone, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "sms", payload, imageOptions(body.Recovery, body.Size))
}

func (s *Server) geolocationQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Label     string  `json:"label"`
		Size      int     `json:"size"`
		Recovery  string  `json:"recovery"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	payload, err := qr.GeolocationPayload(body.Latitude, body.Longitude, body.Label)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "geolocation", payload, imageOptions(body.Recovery, body.Size))
}
