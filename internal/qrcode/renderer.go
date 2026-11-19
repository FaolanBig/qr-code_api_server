package qrcode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	lib "github.com/skip2/go-qrcode"
)

type ImageOptions struct {
	Size       int
	Recovery   string
	Foreground string
	Background string
	Logo       []byte
	LogoSource string
}

type Result struct {
	Type         string `json:"type"`
	Format       string `json:"format"`
	Size         int    `json:"size"`
	Recovery     string `json:"recovery"`
	LogoIncluded bool   `json:"logo_included"`
	LogoSource   string `json:"logo_source,omitempty"`
	PNGBase64    string `json:"png_base64"`
	DataURI      string `json:"data_uri"`
}

func Generate(payloadType, payload string, options ImageOptions) (Result, error) {
	if options.Size == 0 {
		options.Size = 256
	}
	if options.Size < 128 || options.Size > 2048 {
		return Result{}, fmt.Errorf("size must be between 128 and 2048 pixels")
	}

	recoveryName := options.Recovery
	if recoveryName == "" {
		recoveryName = "medium"
	}
	var level lib.RecoveryLevel
	switch recoveryName {
	case "low":
		level = lib.Low
	case "medium":
		level = lib.Medium
	case "high":
		level = lib.High
	case "highest":
		level = lib.Highest
	default:
		return Result{}, fmt.Errorf("recovery must be low, medium, high, or highest")
	}

	pngBytes, err := lib.Encode(payload, level, options.Size)
	if err != nil {
		return Result{}, fmt.Errorf("unable to encode QR code: %w", err)
	}

	logoSource := ""
	if len(options.Logo) > 0 {
		pngBytes, err = overlayLogo(pngBytes, options.Logo, options.Size)
		if err != nil {
			return Result{}, err
		}
		logoSource = options.LogoSource
		if logoSource == "" {
			logoSource = "custom"
		}
	}

	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	return Result{
		Type:         payloadType,
		Format:       "png",
		Size:         options.Size,
		Recovery:     recoveryName,
		LogoIncluded: len(options.Logo) > 0,
		LogoSource:   logoSource,
		PNGBase64:    encoded,
		DataURI:      "data:image/png;base64," + encoded,
	}, nil
}

func ValidatePNGLogo(data []byte, maxBytes int) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("logo is empty")
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("logo exceeds maximum size of %d bytes", maxBytes)
	}
	if len(data) >= 8 {
		signature := []byte("\x89PNG\r\n\x1a\n")
		if !bytes.Equal(data[:8], signature) {
			return nil, fmt.Errorf("logo must be a PNG file")
		}
	} else {
		return nil, fmt.Errorf("logo must be a PNG file")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("logo is not a valid PNG image")
	}
	bounds := img.Bounds()
	if bounds.Dx() < 16 || bounds.Dy() < 16 {
		return nil, fmt.Errorf("logo dimensions must be at least 16x16 pixels")
	}
	if bounds.Dx() > 1024 || bounds.Dy() > 1024 {
		return nil, fmt.Errorf("logo dimensions must not exceed 1024x1024 pixels")
	}
	return img, nil
}

func DecodeBase64PNG(value string, maxBytes int) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("logo_png_base64 is empty")
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("logo_png_base64 is not valid base64")
	}
	if _, err := ValidatePNGLogo(data, maxBytes); err != nil {
		return nil, err
	}
	return data, nil
}

func overlayLogo(qrBytes, logoBytes []byte, qrSize int) ([]byte, error) {
	qrImage, err := png.Decode(bytes.NewReader(qrBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to decode generated QR code")
	}
	logo, err := ValidatePNGLogo(logoBytes, 10*1024*1024)
	if err != nil {
		return nil, err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, qrSize, qrSize))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), qrImage, qrImage.Bounds().Min, draw.Over)

	logoSize := qrSize / 5
	if logoSize < 24 {
		logoSize = 24
	}
	if logoSize > qrSize/4 {
		logoSize = qrSize / 4
	}
	padding := qrSize / 40
	if padding < 4 {
		padding = 4
	}

	outerSize := logoSize + 2*padding
	centerX, centerY := qrSize/2, qrSize/2
	outer := image.Rect(centerX-outerSize/2, centerY-outerSize/2, centerX+(outerSize+1)/2, centerY+(outerSize+1)/2)
	draw.Draw(canvas, outer, &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)

	logoBounds := image.Rect(centerX-logoSize/2, centerY-logoSize/2, centerX+(logoSize+1)/2, centerY+(logoSize+1)/2)
	resizeNearest(canvas, logoBounds, logo)

	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("unable to encode QR code with logo")
	}
	return output.Bytes(), nil
}

func resizeNearest(dst *image.RGBA, bounds image.Rectangle, src image.Image) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			srcX := src.Bounds().Min.X + (x-bounds.Min.X)*(src.Bounds().Dx()-1)/(bounds.Dx()-1)
			srcY := src.Bounds().Min.Y + (y-bounds.Min.Y)*(src.Bounds().Dy()-1)/(bounds.Dy()-1)
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
}
