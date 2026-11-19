package qrcode

import (
	"encoding/base64"
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

func TestGenerateValidation(t *testing.T) {
	if _, err := Generate("text", "Hello", ImageOptions{Size: 32, Recovery: "medium"}); err == nil {
		t.Fatal("expected size to be rejected")
	}
	if _, err := Generate("text", "Hello", ImageOptions{Size: 256, Recovery: "extreme"}); err == nil {
		t.Fatal("expected recovery level to be rejected")
	}
}
