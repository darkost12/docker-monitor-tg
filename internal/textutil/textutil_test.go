package textutil

import "testing"

func TestTail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short text does not change", "a\nb", 10, "a\nb"},
		{"as a limit", "abcde", 5, "abcde"},
		{"clipped string dropped", "line1\nline2\nline3", 8, "…\nline3"},
		{"no new line", "abcdefghij", 4, "…\nghij"},
		{"cyrillic is read in symbols, not bytes", "привет\nмир", 5, "…\nмир"},
		{"empty string", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tail(tt.in, tt.max); got != tt.want {
				t.Errorf("Tail(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestTailKeepsValidUTF8(t *testing.T) {
	in := "こんにちは"
	got := Tail(in, 3)
	if got != "…\nにちは" {
		t.Errorf("Tail = %q, want %q", got, "…\nにちは")
	}
}

func TestHead(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short text does not change", "abc", 10, "abc"},
		{"as a limit", "abc", 3, "abc"},
		{"clipping", "abcdef", 3, "abc\n…"},
		{"cyrillic", "привет", 3, "при\n…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Head(tt.in, tt.max); got != tt.want {
				t.Errorf("Head(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text does not change", "plain text", "plain text"},
		{"color and reset", "\x1b[96mINFO\x1b[0m", "INFO"},
		{"erase line", "\x1b[2Kline", "line"},
		{"several parameters", "\x1b[1;31mERROR\x1b[0m", "ERROR"},
		{"256 colors", "\x1b[38;5;208mwarn\x1b[0m", "warn"},
		{"cursor control", "\x1b[?25lhidden\x1b[?25h", "hidden"},
		{
			"line from real logs",
			"\x1b[2K\x1b[0m\x1b[2K[\x1b[96m2026-11-12 05:00:22\x1b[0m | \x1b[96mINFO\x1b[0m] [hello] Refreshing stats\x1b[0m",
			"[2026-11-12 05:00:22 | INFO] [hello] Refreshing stats",
		},
		{"brackets without ESC are kept", "array[0] and [96m", "array[0] and [96m"},
		{"cyrillic is kept", "\x1b[32mготово\x1b[0m", "готово"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripANSI(tt.in); got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
