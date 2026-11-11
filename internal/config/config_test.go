package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParse(t *testing.T) {
	t.Run("correct values", func(t *testing.T) {
		cfg, err := Parse(env(map[string]string{
			"TELEGRAM_BOT_TOKEN": "123:abc",
			"ALLOWED_USER_ID":    "42",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Token != "123:abc" || cfg.AllowedUserID != 42 {
			t.Errorf("got %+v", cfg)
		}
	})

	errCases := []struct {
		name    string
		vars    map[string]string
		wantErr string
	}{
		{"no token", map[string]string{"ALLOWED_USER_ID": "42"}, "TELEGRAM_BOT_TOKEN"},
		{"no ID", map[string]string{"TELEGRAM_BOT_TOKEN": "x"}, "ALLOWED_USER_ID"},
		{"nothing set", map[string]string{}, "TELEGRAM_BOT_TOKEN"},
		{"ID not a number", map[string]string{"TELEGRAM_BOT_TOKEN": "x", "ALLOWED_USER_ID": "abc"}, "number"},
	}
	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(env(tt.vars))
			if err == nil {
				t.Fatal("error was expected")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
