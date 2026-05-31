package rsync233

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncCopiesDirectoryContentsWithTrailingSlash(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "hello")
	mustWrite(t, filepath.Join(src, "nested", "b.txt"), "world")

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.CopiedFiles != 2 {
		t.Fatalf("copied files = %d, want 2", summary.CopiedFiles)
	}
	assertFile(t, filepath.Join(dst, "a.txt"), "hello")
	assertFile(t, filepath.Join(dst, "nested", "b.txt"), "world")
}

func TestSyncCopiesDirectoryWithoutTrailingSlashAsChild(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	_, err := Sync(context.Background(), src, dst, Options{Archive: true})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "src", "a.txt"), "hello")
}

func TestSyncDeleteAndExclude(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(src, "cache", "ignored.tmp"), "ignore")
	mustWrite(t, filepath.Join(dst, "old.txt"), "old")

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{
		Archive:  true,
		Delete:   true,
		Excludes: []string{"cache/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.DeletedEntries != 1 {
		t.Fatalf("deleted entries = %d, want 1", summary.DeletedEntries)
	}
	if _, err := os.Stat(filepath.Join(dst, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old file still exists or unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "cache")); !os.IsNotExist(err) {
		t.Fatalf("excluded cache should not be copied: %v", err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
}

func TestCheckReportsDifferenceWithoutWriting(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "new")
	mustWrite(t, filepath.Join(dst, "a.txt"), "old")

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, Check: true, Checksum: true})
	if err != nil {
		t.Fatal(err)
	}
	if !summary.Changed() || summary.UpdatedFiles != 1 {
		t.Fatalf("expected check diff, got %+v", summary)
	}
	assertFile(t, filepath.Join(dst, "a.txt"), "old")
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
