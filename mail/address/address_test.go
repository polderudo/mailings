package address

import "testing"

func TestClean(t *testing.T) {
	cases := map[string]string{
		"gs.\u200bwesendorf@t-online.de":    "gs.wesendorf@t-online.de",
		"info\u200e@gehrenberg-bodensee.de": "info@gehrenberg-bodensee.de",
		"kita\u00adbonho@sjb.karlsruhe.de":  "kitabonho@sjb.karlsruhe.de",
		"ap-grundschule\u00a0@peiting.de":   "ap-grundschule@peiting.de",
		"\ufeffa@b.de\t":                    "a@b.de",
		" x@y.de ":                          "x@y.de",
		"normal@example.com":                "normal@example.com",
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidLocalPart(t *testing.T) {
	ok := []string{"gs.wesendorf", "a", "a+b", "o'brien", "x_y-z", "KiTa-Neuhausen"}
	bad := []string{"", ".a", "a.", "a..b", "gs.\u200bwesendorf", "klappermühle", "ínfo", "a b", "a@b"}
	for _, s := range ok {
		if !ValidLocalPart(s) {
			t.Errorf("ValidLocalPart(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if ValidLocalPart(s) {
			t.Errorf("ValidLocalPart(%q) = true, want false", s)
		}
	}
}

func TestValidLocalPartOf(t *testing.T) {
	if !ValidLocalPartOf("gs.wesendorf@t-online.de") {
		t.Error("saubere Adresse abgelehnt")
	}
	if ValidLocalPartOf("gs.\u200bwesendorf@t-online.de") {
		t.Error("Adresse mit Zero-Width-Space akzeptiert")
	}
	if ValidLocalPartOf("kein-at") {
		t.Error("Adresse ohne @ akzeptiert")
	}
}
