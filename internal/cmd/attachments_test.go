package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "photo.png", "photo.png"},
		{"with spaces", "  image.jpg  ", "image.jpg"},
		{"empty falls back", "", "attachment-7"},
		{"dot falls back", ".", "attachment-7"},
		{"dotdot falls back", "..", "attachment-7"},
		{"forward slash falls back", "../etc/passwd", "attachment-7"},
		{"backslash falls back", `..\windows\system32`, "attachment-7"},
		{"trailing slash falls back", "subdir/", "attachment-7"},
		{"leading slash falls back", "/abs/path", "attachment-7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeFilename(tc.in, 7); got != tc.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUniquePathNoCollision(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.pdf")
	if got := uniquePath(p); got != p {
		t.Errorf("uniquePath = %q, want %q (file does not exist yet)", got, p)
	}
}

func TestUniquePathCollisionAppendsSuffix(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "report.pdf")
	if err := os.WriteFile(existing, []byte("a"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := uniquePath(existing)
	want := filepath.Join(dir, "report-1.pdf")
	if got != want {
		t.Errorf("uniquePath = %q, want %q", got, want)
	}
}

func TestUniquePathCollisionMultiple(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"report.pdf", "report-1.pdf", "report-2.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("a"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	got := uniquePath(filepath.Join(dir, "report.pdf"))
	want := filepath.Join(dir, "report-3.pdf")
	if got != want {
		t.Errorf("uniquePath = %q, want %q", got, want)
	}
}

func TestUniquePathNoExtension(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "Makefile")
	if err := os.WriteFile(existing, []byte("a"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := uniquePath(existing)
	want := filepath.Join(dir, "Makefile-1")
	if got != want {
		t.Errorf("uniquePath = %q, want %q", got, want)
	}
}
