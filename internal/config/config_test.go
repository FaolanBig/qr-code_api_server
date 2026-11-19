package config

import (
	"testing"
)

func TestListenAddressFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "port only", value: "8080", want: ":8080"},
		{name: "host port", value: "127.0.0.1:8080", want: "127.0.0.1:8080"},
		{name: "colon port", value: ":80", want: ":80"},
		{name: "empty defaults", value: "", want: ":8080"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LISTEN_ADDRESS", test.value)
			got, err := listenAddressFromEnv("LISTEN_ADDRESS", ":8080")
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestListenAddressFromEnvRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "65536", "localhost", "localhost:not-a-port", "127.0.0.1:"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("LISTEN_ADDRESS", value)
			if _, err := listenAddressFromEnv("LISTEN_ADDRESS", ":8080"); err == nil {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}
}
