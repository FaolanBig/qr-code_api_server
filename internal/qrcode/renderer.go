package qrcode

import (
	"encoding/base64"
	"fmt"

	lib "github.com/skip2/go-qrcode"
)

type ImageOptions struct {
	Size       int
	Recovery   string
	Foreground string
	Background string
}

type Result struct {
	Type      string `json:"type"`
	Format    string `json:"format"`
	Size      int    `json:"size"`
	Recovery  string `json:"recovery"`
	PNGBase64 string `json:"png_base64"`
	DataURI   string `json:"data_uri"`
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

	png, err := lib.Encode(payload, level, options.Size)
	if err != nil {
		return Result{}, fmt.Errorf("unable to encode QR code: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(png)
	return Result{
		Type:      payloadType,
		Format:    "png",
		Size:      options.Size,
		Recovery:  recoveryName,
		PNGBase64: encoded,
		DataURI:   "data:image/png;base64," + encoded,
	}, nil
}
