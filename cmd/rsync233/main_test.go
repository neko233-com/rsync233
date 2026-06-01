package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDefaultDoesNotRecurseIntoDirectories(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	var stdout, stderr bytes.Buffer
	err := run([]string{src + string(os.PathSeparator), dst}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected directory recursion error")
	}
	if !strings.Contains(err.Error(), "use -r or -a") {
		t.Fatalf("error = %q, want recursion hint", err)
	}
}

func TestRunArchiveRecursesIntoDirectories(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-a", src + string(os.PathSeparator), dst}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("copied file = %q, want hello", got)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
