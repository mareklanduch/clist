package tui

import "testing"

func TestWrapFitsWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		w    int
	}{
		{"plain words", "buy milk and eggs at the store", 10},
		{"single overlong word", "https://example.com/a/very/long/unbroken/url/that/never/ends", 12},
		{"overlong word between words", "see https://example.com/extremely/long/path now", 10},
		{"unicode", "žlutý kůň úpěl ďábelské ódy příšerně", 8},
		{"empty", "", 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			segs := wrap(c.in, c.w)
			if len(segs) == 0 {
				t.Fatal("wrap returned no segments")
			}
			for _, s := range segs {
				if n := len([]rune(s)); n > c.w {
					t.Errorf("segment %q is %d runes, exceeds width %d", s, n, c.w)
				}
			}
		})
	}
}

func TestTruncateEllipsis(t *testing.T) {
	if got := truncateEllipsis("héllo wörld", 8); len([]rune(got)) > 8 {
		t.Errorf("truncateEllipsis exceeded width: %q", got)
	}
	if got := truncateEllipsis("short", 10); got != "short" {
		t.Errorf("unexpected truncation: %q", got)
	}
}
