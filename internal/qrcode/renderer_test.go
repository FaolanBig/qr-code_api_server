package qrcode

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	result, err := Generate("text", "Hello", ImageOptions{Size: 256, Recovery: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Format != "png" || result.Size != 256 {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	if !strings.HasPrefix(result.DataURI, "data:image/png;base64,") {
		t.Fatalf("unexpected data URI: %q", result.DataURI[:20])
	}
	png, err := base64.StdEncoding.DecodeString(result.PNGBase64)
	if err != nil || len(png) < 8 {
		t.Fatalf("invalid PNG base64: len=%d err=%v", len(png), err)
	}
	if string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("generated data is not a PNG")
	}
}

func makeTestPNG(t *testing.T) []byte {
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

func TestGenerateWithLogo(t *testing.T) {
	logo := makeTestPNG(t)
	result, err := Generate("text", "Hello", ImageOptions{Size: 256, Recovery: "highest", Logo: logo, LogoSource: "custom"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.LogoIncluded || result.LogoSource != "custom" {
		t.Fatalf("unexpected logo metadata: %+v", result)
	}
}

func TestValidatePNGLogo(t *testing.T) {
	logo := makeTestPNG(t)
	if _, err := ValidatePNGLogo(logo, 1024*1024); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidatePNGLogo([]byte("not-png"), 1024*1024); err == nil {
		t.Fatal("expected invalid PNG to fail")
	}
	if _, err := DecodeBase64PNG(base64.StdEncoding.EncodeToString(logo), 1024*1024); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateValidation(t *testing.T) {
	if _, err := Generate("text", "Hello", ImageOptions{Size: 32, Recovery: "medium"}); err == nil {
		t.Fatal("expected size to be rejected")
	}
	if _, err := Generate("text", "Hello", ImageOptions{Size: 256, Recovery: "extreme"}); err == nil {
		t.Fatal("expected recovery level to be rejected")
	}
}
