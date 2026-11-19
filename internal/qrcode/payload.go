package qrcode

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var numberPattern = regexp.MustCompile(`^[+0-9][0-9() .-]*$`)

func EscapeText(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\r\n", "\\n", "\n", "\\n", "\r", "\\n")
	return replacer.Replace(value)
}

func URLPayload(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("url is required")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("url must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("url scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("url must include a host")
	}
	return parsed.String(), nil
}

func VCardPayload(firstName, lastName, organization, title, phone, email, website, address string) (string, error) {
	if strings.TrimSpace(firstName) == "" && strings.TrimSpace(lastName) == "" {
		return "", fmt.Errorf("at least first_name or last_name is required")
	}
	var lines []string
	lines = append(lines, "BEGIN:VCARD", "VERSION:3.0")
	lines = append(lines, "N:"+EscapeText(lastName)+";"+EscapeText(firstName)+";;;")
	lines = append(lines, "FN:"+EscapeText(strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))))
	if organization != "" {
		lines = append(lines, "ORG:"+EscapeText(organization))
	}
	if title != "" {
		lines = append(lines, "TITLE:"+EscapeText(title))
	}
	if phone != "" {
		lines = append(lines, "TEL;TYPE=CELL:"+EscapeText(phone))
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return "", fmt.Errorf("email is invalid")
		}
		lines = append(lines, "EMAIL;TYPE=INTERNET:"+EscapeText(email))
	}
	if website != "" {
		if _, err := URLPayload(website); err != nil {
			return "", fmt.Errorf("website is invalid")
		}
		lines = append(lines, "URL:"+EscapeText(website))
	}
	if address != "" {
		lines = append(lines, "ADR;TYPE=WORK:;;"+EscapeText(address)+";;;;")
	}
	lines = append(lines, "END:VCARD")
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

func WiFiPayload(ssid, password, security string, hidden bool) (string, error) {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return "", fmt.Errorf("ssid is required")
	}
	security = strings.ToUpper(strings.TrimSpace(security))
	if security == "" {
		security = "WPA"
	}
	if security != "WPA" && security != "WEP" && security != "NOPASS" {
		return "", fmt.Errorf("security must be WPA, WEP, or NOPASS")
	}
	if security == "NOPASS" {
		password = ""
	}
	if security != "NOPASS" && strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password is required for secured Wi-Fi networks")
	}
	escape := func(value string) string {
		return strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", ":", "\\:").Replace(value)
	}
	payload := "WIFI:T:" + security + ";S:" + escape(ssid) + ";P:" + escape(password) + ";"
	if hidden {
		payload += "H:true;"
	}
	payload += ";"
	return payload, nil
}

func CalendarPayload(title, description, location, start, end, timezone string, allDay bool) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	var startValue, endValue string
	if allDay {
		startTime, err := time.Parse("2006-01-02", start)
		if err != nil {
			return "", fmt.Errorf("start must use YYYY-MM-DD for an all-day event")
		}
		startValue = ";VALUE=DATE:" + startTime.Format("20060102")
		if end == "" {
			endValue = ";VALUE=DATE:" + startTime.AddDate(0, 0, 1).Format("20060102")
		} else {
			endTime, err := time.Parse("2006-01-02", end)
			if err != nil {
				return "", fmt.Errorf("end must use YYYY-MM-DD for an all-day event")
			}
			if !endTime.After(startTime) {
				return "", fmt.Errorf("end must be after start")
			}
			endValue = ";VALUE=DATE:" + endTime.Format("20060102")
		}
	} else {
		startTime, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return "", fmt.Errorf("start must use RFC3339 format")
		}
		if end == "" {
			endTime := startTime.Add(time.Hour)
			endValue = ":" + endTime.UTC().Format("20060102T150405Z")
		} else {
			endTime, err := time.Parse(time.RFC3339, end)
			if err != nil {
				return "", fmt.Errorf("end must use RFC3339 format")
			}
			if !endTime.After(startTime) {
				return "", fmt.Errorf("end must be after start")
			}
			endValue = ":" + endTime.UTC().Format("20060102T150405Z")
		}
		startValue = ":" + startTime.UTC().Format("20060102T150405Z")
	}

	stamp := time.Now().UTC().Format("20060102T150405Z")
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//QR Code API//EN", "BEGIN:VEVENT", "UID:" + stamp + "@qrcode-api", "DTSTAMP:" + stamp}
	lines = append(lines, "DTSTART"+startValue)
	lines = append(lines, "DTEND"+endValue)
	lines = append(lines, "SUMMARY:"+EscapeText(title))
	if description != "" {
		lines = append(lines, "DESCRIPTION:"+EscapeText(description))
	}
	if location != "" {
		lines = append(lines, "LOCATION:"+EscapeText(location))
	}
	if timezone != "" {
		lines = append(lines, "X-WR-TIMEZONE:"+EscapeText(timezone))
	}
	lines = append(lines, "END:VEVENT", "END:VCALENDAR")
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

func TextPayload(text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text is required")
	}
	return text, nil
}

func FilePayload(filename, mimeType, encoded string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if strings.ContainsAny(filename, "\r\n") {
		return "", fmt.Errorf("filename must not contain line breaks")
	}
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if strings.ContainsAny(mimeType, "\r\n") {
		return "", fmt.Errorf("mime_type must not contain line breaks")
	}
	if len(encoded) > 4096 {
		return "", fmt.Errorf("base64 content is too large")
	}
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64_content is not valid base64")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("file content is empty")
	}
	if len(content) > 1024 {
		return "", fmt.Errorf("decoded file content exceeds 1024 bytes")
	}
	return "data:" + mimeType + ";name=" + url.QueryEscape(filename) + ";base64," + base64.StdEncoding.EncodeToString(content), nil
}

func NumberPayload(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("number is required")
	}
	if !numberPattern.MatchString(value) {
		return "", fmt.Errorf("number contains unsupported characters")
	}
	hasDigit := false
	for _, allowed := range value {
		if allowed >= '0' && allowed <= '9' {
			hasDigit = true
			break
		}
	}
	if !hasDigit {
		return "", fmt.Errorf("number must contain at least one digit")
	}
	return value, nil
}

func EmailPayload(address, subject, body string) (string, error) {
	if _, err := mail.ParseAddress(address); err != nil {
		return "", fmt.Errorf("email is invalid")
	}
	values := url.Values{}
	if subject != "" {
		values.Set("subject", subject)
	}
	if body != "" {
		values.Set("body", body)
	}
	payload := "mailto:" + address
	if encoded := values.Encode(); encoded != "" {
		payload += "?" + encoded
	}
	return payload, nil
}

func SMSPayload(phone, body string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "", fmt.Errorf("phone is required")
	}
	if strings.ContainsAny(phone, "\r\n") {
		return "", fmt.Errorf("phone must not contain line breaks")
	}
	payload := "smsto:" + phone
	if body != "" {
		payload += ":" + body
	}
	return payload, nil
}

func GeolocationPayload(latitude, longitude float64, label string) (string, error) {
	if latitude < -90 || latitude > 90 {
		return "", fmt.Errorf("latitude must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return "", fmt.Errorf("longitude must be between -180 and 180")
	}
	payload := fmt.Sprintf("geo:%.6f,%.6f", latitude, longitude)
	if strings.TrimSpace(label) != "" {
		payload += "?q=" + url.QueryEscape(label)
	}
	return payload, nil
}
