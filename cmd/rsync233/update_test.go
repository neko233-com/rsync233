package main

import "testing"

func TestFormatVersionTag(t *testing.T) {
	for input, want := range map[string]string{
		"1.2.3":  "v1.2.3",
		"v1.2.3": "v1.2.3",
		"V1.2.3": "v1.2.3",
		"":       "v0.0.0",
	} {
		if got := formatVersionTag(input); got != want {
			t.Fatalf("formatVersionTag(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.2.0", "v1.10.0", true},
		{"v2.0.0", "v1.9.9", false},
		{"v1.0.0", "v1.0.0", false},
	}
	for _, tt := range tests {
		if got := versionLess(tt.a, tt.b); got != tt.want {
			t.Fatalf("versionLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestReleaseAssetURL(t *testing.T) {
	got := releaseAssetURL("v1.2.3", "windows", "amd64")
	want := "https://github.com/neko233-com/rsync233/releases/download/v1.2.3/rsync233-windows-amd64.exe"
	if got != want {
		t.Fatalf("releaseAssetURL windows = %q, want %q", got, want)
	}

	got = releaseAssetURL("1.2.3", "linux", "arm64")
	want = "https://github.com/neko233-com/rsync233/releases/download/v1.2.3/rsync233-linux-arm64"
	if got != want {
		t.Fatalf("releaseAssetURL linux = %q, want %q", got, want)
	}
}
