package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	qr "github.com/FaolanBig/qr-code_api_server/internal/qrcode"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const maxRequestBytes = 12 << 20

type logoFields struct {
	ShowLogo      bool   `json:"show_logo,omitempty"`
	LogoPNGBase64 string `json:"logo_png_base64,omitempty"`
}

type Server struct {
	apiKey       string
	fallbackLogo []byte
	logoMaxBytes int
	openAPISpec  []byte
	logger       *slog.Logger
}

func New(apiKey string, logger *slog.Logger, options ...ServerOption) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	server := &Server{apiKey: apiKey, logoMaxBytes: 1024 * 1024, logger: logger}
	for _, option := range options {
		option(server)
	}
	return server
}

type ServerOption func(*Server)

func WithFallbackLogo(data []byte) ServerOption {
	return func(server *Server) {
		server.fallbackLogo = data
	}
}

func WithOpenAPISpec(spec []byte) ServerOption {
	return func(server *Server) {
		server.openAPISpec = spec
	}
}

func WithLogoMaxBytes(maxBytes int) ServerOption {
	return func(server *Server) {
		if maxBytes > 0 {
			server.logoMaxBytes = maxBytes
		}
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(securityHeaders)
	r.Use(requestLogger(s.logger))
	r.Use(middleware.Recoverer)

	r.Get("/", s.webIndex)
	r.Get("/app.css", s.webStyles)
	r.Get("/app.js", s.webScript)
	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)
	r.Get("/openapi.yaml", s.openAPI)

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

func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	if len(s.openAPISpec) == 0 {
		writeError(w, http.StatusNotFound, "OpenAPI specification is not configured")
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(s.openAPISpec)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.openAPISpec)
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) apiIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "QR Code API",
		"version": "v2.0.0-rc",
		"endpoints": []string{
			"POST /api/v1/qr/url", "POST /api/v1/qr/vcard", "POST /api/v1/qr/wifi",
			"POST /api/v1/qr/calendar", "POST /api/v1/qr/text", "POST /api/v1/qr/file",
			"POST /api/v1/qr/number", "POST /api/v1/qr/email", "POST /api/v1/qr/sms",
			"POST /api/v1/qr/geolocation",
		},
	})
}

func decodeRequest(w http.ResponseWriter, r *http.Request, destination any) ([]byte, bool) {
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		return decodeMultipart(w, r, destination)
	}
	if !decodeJSON(w, r, destination) {
		return nil, false
	}
	return nil, true
}

func decodeMultipart(w http.ResponseWriter, r *http.Request, destination any) ([]byte, bool) {
	const bodyLimit = int64(maxRequestBytes)
	r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)
	if err := r.ParseMultipartForm(bodyLimit); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart request: "+err.Error())
		return nil, false
	}

	for name, values := range r.MultipartForm.Value {
		if len(values) == 0 {
			continue
		}
		if !setFormField(reflect.ValueOf(destination).Elem(), name, values[0]) {
			writeError(w, http.StatusBadRequest, "unknown multipart field: "+name)
			return nil, false
		}
	}

	var logo []byte
	for _, fieldName := range []string{"logo", "logo_file"} {
		fileHeaders := r.MultipartForm.File[fieldName]
		if len(fileHeaders) == 0 {
			continue
		}
		if len(fileHeaders) != 1 {
			writeError(w, http.StatusBadRequest, "exactly one logo file is allowed")
			return nil, false
		}
		file, err := fileHeaders[0].Open()
		if err != nil {
			writeError(w, http.StatusBadRequest, "unable to open uploaded logo")
			return nil, false
		}
		logo, err = io.ReadAll(io.LimitReader(file, 10*1024*1024+1))
		file.Close()
		if err != nil {
			writeError(w, http.StatusBadRequest, "unable to read uploaded logo")
			return nil, false
		}
		if len(logo) > 10*1024*1024 {
			writeError(w, http.StatusBadRequest, "logo exceeds maximum size")
			return nil, false
		}
		break
	}
	return logo, true
}

func setFormField(target reflect.Value, name, value string) bool {
	if target.Kind() == reflect.Pointer {
		return setFormField(target.Elem(), name, value)
	}
	if target.Kind() != reflect.Struct {
		return false
	}
	targetType := target.Type()
	for i := 0; i < target.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := target.Field(i)
		if field.Anonymous {
			if setFormField(fieldValue, name, value) {
				return true
			}
			continue
		}
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" || tag != name {
			continue
		}
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(value)
		case reflect.Bool:
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return false
			}
			fieldValue.SetBool(parsed)
		case reflect.Int:
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return false
			}
			fieldValue.SetInt(int64(parsed))
		case reflect.Float64:
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return false
			}
			fieldValue.SetFloat(parsed)
		default:
			return false
		}
		return true
	}
	return false
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

func (s *Server) imageOptions(recovery string, size int, showLogo bool, logoBase64 string, multipartLogo []byte) (qr.ImageOptions, error) {
	if strings.TrimSpace(recovery) == "" {
		recovery = "medium"
	}
	options := qr.ImageOptions{
		Size:     size,
		Recovery: strings.ToLower(strings.TrimSpace(recovery)),
	}
	if !showLogo {
		return options, nil
	}

	if len(multipartLogo) > 0 {
		if _, err := qr.ValidatePNGLogo(multipartLogo, s.logoMaxBytes); err != nil {
			return qr.ImageOptions{}, err
		}
		options.Logo = multipartLogo
		options.LogoSource = "custom"
		return options, nil
	}
	if strings.TrimSpace(logoBase64) != "" {
		logo, err := qr.DecodeBase64PNG(logoBase64, s.logoMaxBytes)
		if err != nil {
			return qr.ImageOptions{}, err
		}
		options.Logo = logo
		options.LogoSource = "custom"
		return options, nil
	}
	if len(s.fallbackLogo) > 0 {
		if _, err := qr.ValidatePNGLogo(s.fallbackLogo, s.logoMaxBytes); err != nil {
			return qr.ImageOptions{}, fmt.Errorf("invalid fallback logo: %w", err)
		}
		options.Logo = s.fallbackLogo
		options.LogoSource = "fallback"
		return options, nil
	}
	return qr.ImageOptions{}, fmt.Errorf("show_logo is enabled but no logo was provided and no fallback logo is configured")
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
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.URLPayload(body.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "url", payload, options)
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
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.VCardPayload(body.FirstName, body.LastName, body.Organization, body.Title, body.Phone, body.Email, body.Website, body.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "vcard", payload, options)
}

func (s *Server) wifiQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
		Security string `json:"security"`
		Hidden   bool   `json:"hidden"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.WiFiPayload(body.SSID, body.Password, body.Security, body.Hidden)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "wifi", payload, options)
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
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.CalendarPayload(body.Title, body.Description, body.Location, body.Start, body.End, body.Timezone, body.AllDay)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "calendar", payload, options)
}

func (s *Server) textQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text     string `json:"text"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.TextPayload(body.Text)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "text", payload, options)
}

func (s *Server) fileQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Filename      string `json:"filename"`
		MIMEType      string `json:"mime_type"`
		Base64Content string `json:"base64_content"`
		Size          int    `json:"size"`
		Recovery      string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.FilePayload(body.Filename, body.MIMEType, body.Base64Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "file", payload, options)
}

func (s *Server) numberQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Number   string `json:"number"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.NumberPayload(body.Number)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "number", payload, options)
}

func (s *Server) emailQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.EmailPayload(body.Email, body.Subject, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "email", payload, options)
}

func (s *Server) smsQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone    string `json:"phone"`
		Body     string `json:"body"`
		Size     int    `json:"size"`
		Recovery string `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.SMSPayload(body.Phone, body.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "sms", payload, options)
}

func (s *Server) geolocationQR(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Label     string  `json:"label"`
		Size      int     `json:"size"`
		Recovery  string  `json:"recovery"`
		logoFields
	}
	multipartLogo, ok := decodeRequest(w, r, &body)
	if !ok {
		return
	}
	options, err := s.imageOptions(body.Recovery, body.Size, body.ShowLogo, body.LogoPNGBase64, multipartLogo)
	if err != nil {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, err := qr.GeolocationPayload(body.Latitude, body.Longitude, body.Label)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	render(w, "geolocation", payload, options)
}
