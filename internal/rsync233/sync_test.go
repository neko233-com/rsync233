package rsync233

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestChecksumComparesEqualSizeFilesEvenWhenTimesDiffer(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	srcFile := filepath.Join(src, "a.txt")
	dstFile := filepath.Join(dst, "a.txt")
	mustWrite(t, srcFile, "new")
	mustWrite(t, dstFile, "old")
	setModTime(t, srcFile, time.Unix(100, 0))
	setModTime(t, dstFile, time.Unix(200, 0))

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, Checksum: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.UpdatedFiles != 1 {
		t.Fatalf("updated files = %d, want 1", summary.UpdatedFiles)
	}
	assertFile(t, dstFile, "new")
}

func TestIgnoreExistingSkipsReceiverFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "new")
	mustWrite(t, filepath.Join(dst, "a.txt"), "old")

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, IgnoreExisting: true, IgnoreTimes: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Changed() {
		t.Fatalf("expected no changes, got %+v", summary)
	}
	assertFile(t, filepath.Join(dst, "a.txt"), "old")
}

func TestExistingSkipsNewReceiverFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "new.txt"), "new")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, Existing: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Changed() {
		t.Fatalf("expected no changes, got %+v", summary)
	}
	if _, err := os.Stat(filepath.Join(dst, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("new receiver file should be skipped: %v", err)
	}
}

func TestUpdateSkipsNewerReceiverFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	srcFile := filepath.Join(src, "a.txt")
	dstFile := filepath.Join(dst, "a.txt")
	mustWrite(t, srcFile, "newer-source")
	mustWrite(t, dstFile, "newer-dest")
	setModTime(t, srcFile, time.Unix(100, 0))
	setModTime(t, dstFile, time.Unix(200, 0))

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, Update: true, IgnoreTimes: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Changed() {
		t.Fatalf("expected no changes, got %+v", summary)
	}
	assertFile(t, dstFile, "newer-dest")
}

func TestSizeOnlySkipsMatchingSizeDespiteDifferentTimes(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	srcFile := filepath.Join(src, "a.txt")
	dstFile := filepath.Join(dst, "a.txt")
	mustWrite(t, srcFile, "abc")
	mustWrite(t, dstFile, "xyz")
	setModTime(t, srcFile, time.Unix(100, 0))
	setModTime(t, dstFile, time.Unix(200, 0))

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true, SizeOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Changed() {
		t.Fatalf("expected no changes, got %+v", summary)
	}
	assertFile(t, dstFile, "xyz")
}

func TestIncludeOverridesExclude(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(src, "drop.log"), "drop")

	_, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{
		Archive:  true,
		Includes: []string{"keep.txt"},
		Excludes: []string{"*.txt", "*.log"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
	if _, err := os.Stat(filepath.Join(dst, "drop.log")); !os.IsNotExist(err) {
		t.Fatalf("drop.log should be excluded: %v", err)
	}
}

func TestArchivePreservesSymlink(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "target.txt"), "target")
	link := filepath.Join(src, "link.txt")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Archive: true})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link.txt mode = %v, want symlink", info.Mode())
	}
	target, err := os.Readlink(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "target.txt" {
		t.Fatalf("link target = %q, want target.txt", target)
	}
}

func TestLinksUpdatesExistingSymlink(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("new.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.Symlink("old.txt", filepath.Join(dst, "link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	summary, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{Recursive: true, Links: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.UpdatedFiles != 1 {
		t.Fatalf("updated files = %d, want 1", summary.UpdatedFiles)
	}
	target, err := os.Readlink(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "new.txt" {
		t.Fatalf("link target = %q, want new.txt", target)
	}
}

func TestDirectoryRequiresRecursiveOrArchive(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	mustWrite(t, filepath.Join(src, "a.txt"), "hello")

	_, err := Sync(context.Background(), src+string(os.PathSeparator), dst, Options{})
	if err == nil {
		t.Fatal("expected directory recursion error")
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

func setModTime(t *testing.T, p string, ts time.Time) {
	t.Helper()
	if err := os.Chtimes(p, ts, ts); err != nil {
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
