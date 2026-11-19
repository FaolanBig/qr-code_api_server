package qrcode

import (
	"strings"
	"testing"
)

func TestURLPayload(t *testing.T) {
	payload, err := URLPayload("https://example.com/path?q=hello")
	if err != nil || payload != "https://example.com/path?q=hello" {
		t.Fatalf("URLPayload() = %q, %v", payload, err)
	}
	if _, err := URLPayload("ftp://example.com"); err == nil {
		t.Fatal("expected unsupported URL scheme to fail")
	}
}

func TestVCardPayload(t *testing.T) {
	payload, err := VCardPayload("Ada", "Lovelace", "Analytical Engines", "Mathematician", "+44 123", "ada@example.com", "https://example.com", "London")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"BEGIN:VCARD", "N:Lovelace;Ada;;;", "EMAIL;TYPE=INTERNET:ada@example.com", "END:VCARD"} {
		if !strings.Contains(payload, expected) {
			t.Errorf("vCard payload missing %q", expected)
		}
	}
}

func TestWiFiPayload(t *testing.T) {
	payload, err := WiFiPayload("Guest;Network", "password:123", "WPA", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, "S:Guest\\;Network") || !strings.Contains(payload, "H:true") {
		t.Fatalf("unexpected Wi-Fi payload: %q", payload)
	}
}

func TestCalendarPayload(t *testing.T) {
	payload, err := CalendarPayload("Review", "Quarterly review", "Office", "2026-12-01T09:30:00Z", "2026-12-01T10:30:00Z", "UTC", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"BEGIN:VEVENT", "DTSTART:20261201T093000Z", "DTEND:20261201T103000Z", "END:VCALENDAR"} {
		if !strings.Contains(payload, expected) {
			t.Errorf("calendar payload missing %q", expected)
		}
	}
}

func TestFileAndNumberPayload(t *testing.T) {
	payload, err := FilePayload("hello.txt", "text/plain", "aGVsbG8=")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, "data:text/plain;name=hello.txt;base64,aGVsbG8=") {
		t.Fatalf("unexpected file payload: %q", payload)
	}
	if _, err := NumberPayload("+1 (555) 123-4567"); err != nil {
		t.Fatal(err)
	}
	if _, err := NumberPayload("hello"); err == nil {
		t.Fatal("expected invalid number to fail")
	}
}

func TestSuggestedTypePayloads(t *testing.T) {
	tests := []struct {
		name   string
		fn     func() (string, error)
		prefix string
	}{
		{"email", func() (string, error) { return EmailPayload("user@example.com", "Hello", "World") }, "mailto:"},
		{"sms", func() (string, error) { return SMSPayload("+15551234567", "Hi") }, "smsto:"},
		{"geolocation", func() (string, error) { return GeolocationPayload(51.5074, -0.1278, "London") }, "geo:"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := test.fn()
			if err != nil || !strings.HasPrefix(payload, test.prefix) {
				t.Fatalf("payload = %q, %v", payload, err)
			}
		})
	}
}
