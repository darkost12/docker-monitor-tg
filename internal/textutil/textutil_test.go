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
