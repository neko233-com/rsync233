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

func TestRunVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "rsync233 ") {
		t.Fatalf("version output = %q", stdout.String())
	}
}

func TestRunExcludeFromFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	excludes := filepath.Join(root, "exclude.txt")
	mustWrite(t, filepath.Join(src, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(src, "drop.tmp"), "drop")
	mustWrite(t, excludes, "# comment\n*.tmp\n")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-a", "--exclude-from", excludes, src + string(os.PathSeparator), dst}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
	if _, err := os.Stat(filepath.Join(dst, "drop.tmp")); !os.IsNotExist(err) {
		t.Fatalf("drop.tmp should be excluded: %v", err)
	}
}

func TestRunIncludeFromFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	includes := filepath.Join(root, "include.txt")
	mustWrite(t, filepath.Join(src, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(src, "drop.txt"), "drop")
	mustWrite(t, includes, "keep.txt\n")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-a", "--include-from", includes, "--exclude", "*.txt", src + string(os.PathSeparator), dst}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
	if _, err := os.Stat(filepath.Join(dst, "drop.txt")); !os.IsNotExist(err) {
		t.Fatalf("drop.txt should be excluded: %v", err)
	}
}

func TestRunFilterRules(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(src, "drop.txt"), "drop")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"-a", "-f", "+ keep.txt", "-f", "- *.txt", src + string(os.PathSeparator), dst}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
	if _, err := os.Stat(filepath.Join(dst, "drop.txt")); !os.IsNotExist(err) {
		t.Fatalf("drop.txt should be excluded: %v", err)
	}
}

func assertFile(t *testing.T, p, want string) {
	t.Helper()
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s = %q, want %q", p, got, want)
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
