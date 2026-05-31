package rsync233

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"time"
)

var ErrDiffFound = errors.New("differences found")

type Options struct {
	Archive       bool
	Delete        bool
	DryRun        bool
	Check         bool
	Checksum      bool
	PreserveOwner bool
	Progress      bool
	Excludes      []string
	Logger        *slog.Logger
}

func Sync(ctx context.Context, sourceRaw, destRaw string, opts Options) (Summary, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Check {
		opts.DryRun = true
	}
	summary := Summary{DryRun: opts.DryRun, CheckOnly: opts.Check}

	srcEP, err := ParseEndpoint(sourceRaw)
	if err != nil {
		return summary, err
	}
	dstEP, err := ParseEndpoint(destRaw)
	if err != nil {
		return summary, err
	}

	srcFS, err := OpenFS(ctx, srcEP)
	if err != nil {
		return summary, err
	}
	defer srcFS.Close()
	dstFS, err := OpenFS(ctx, dstEP)
	if err != nil {
		return summary, err
	}
	defer dstFS.Close()

	srcRoot := srcEP.Path
	dstRoot := dstEP.Path
	srcInfo, err := srcFS.Stat(ctx, srcRoot)
	if err != nil {
		return summary, fmt.Errorf("stat source: %w", err)
	}
	if srcInfo.IsDir && !srcEP.Trailing {
		dstRoot = joinPath(dstEP.IsRemote(), dstRoot, endpointBase(srcRoot))
	}

	excluder := NewExcluder(opts.Excludes)
	sourceIndex := map[string]FileInfo{}

	err = srcFS.Walk(ctx, srcRoot, func(srcPath string, info FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := relPath(srcEP.IsRemote(), srcRoot, srcPath)
		if err != nil {
			return err
		}
		rel = normalizeRel(rel)
		if excluder.Match(rel, info.IsDir) {
			summary.SkippedEntries++
			if info.IsDir {
				return fs.SkipDir
			}
			return nil
		}
		sourceIndex[rel] = info
		if info.IsDir {
			summary.ScannedDirs++
		} else {
			summary.ScannedFiles++
		}
		dstPath := dstRoot
		if rel != "." {
			dstPath = joinPath(dstEP.IsRemote(), dstRoot, rel)
		}
		if info.IsDir {
			created, err := ensureDir(ctx, dstFS, dstPath, info.Mode, opts)
			if err != nil {
				return err
			}
			if created {
				summary.CreatedDirs++
				opts.Logger.Info("mkdir", "path", dstPath)
			}
			return nil
		}
		copied, updated, bytesCopied, err := syncFile(ctx, srcFS, dstFS, srcPath, dstPath, info, opts)
		if err != nil {
			return err
		}
		if copied {
			summary.CopiedFiles++
			opts.Logger.Info("copy", "path", dstPath)
		}
		if updated {
			summary.UpdatedFiles++
			opts.Logger.Info("update", "path", dstPath)
		}
		summary.BytesCopied += bytesCopied
		return nil
	})
	if err != nil {
		return summary, err
	}

	if opts.Delete {
		deleted, err := deleteExtraneous(ctx, dstFS, dstRoot, dstEP.IsRemote(), sourceIndex, excluder, opts)
		if err != nil {
			return summary, err
		}
		summary.DeletedEntries += deleted
	}

	return summary, nil
}

func ensureDir(ctx context.Context, dst FileSystem, dstPath string, mode fs.FileMode, opts Options) (bool, error) {
	_, err := dst.Stat(ctx, dstPath)
	if err == nil {
		if opts.Archive && !opts.DryRun {
			_ = dst.Chmod(ctx, dstPath, mode.Perm())
		}
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if opts.DryRun {
		return true, nil
	}
	return true, dst.MkdirAll(ctx, dstPath, mode.Perm())
}

func syncFile(ctx context.Context, src, dst FileSystem, srcPath, dstPath string, srcInfo FileInfo, opts Options) (copied bool, updated bool, bytesCopied int64, err error) {
	dstInfo, err := dst.Stat(ctx, dstPath)
	if err == nil && dstInfo.IsDir {
		return false, false, 0, fmt.Errorf("destination %s is a directory", dstPath)
	}
	needsCopy := false
	isCreate := false
	if errors.Is(err, os.ErrNotExist) {
		needsCopy = true
		isCreate = true
	} else if err != nil {
		return false, false, 0, err
	} else if dstInfo.Size != srcInfo.Size || !sameModTime(dstInfo.ModTime, srcInfo.ModTime) {
		needsCopy = true
	} else if opts.Checksum {
		equal, err := sameChecksum(ctx, src, dst, srcPath, dstPath)
		if err != nil {
			return false, false, 0, err
		}
		needsCopy = !equal
	}
	if !needsCopy {
		return false, false, 0, nil
	}
	if opts.Check {
		return isCreate, !isCreate, 0, nil
	}
	if opts.DryRun {
		return isCreate, !isCreate, 0, nil
	}
	if err := copyFile(ctx, src, dst, srcPath, dstPath, srcInfo.Mode.Perm()); err != nil {
		return false, false, 0, err
	}
	if opts.Archive {
		_ = dst.Chmod(ctx, dstPath, srcInfo.Mode.Perm())
		_ = dst.Chtimes(ctx, dstPath, srcInfo.ModTime)
	}
	return isCreate, !isCreate, srcInfo.Size, nil
}

func copyFile(ctx context.Context, src, dst FileSystem, srcPath, dstPath string, mode fs.FileMode) error {
	r, err := src.OpenRead(ctx, srcPath)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := dst.OpenWrite(ctx, dstPath, mode)
	if err != nil {
		return err
	}
	defer w.Close()
	buf := make([]byte, 1024*256)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := r.Read(buf)
		if n > 0 {
			if _, err := w.Write(buf[:n]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func sameChecksum(ctx context.Context, src, dst FileSystem, srcPath, dstPath string) (bool, error) {
	a, err := hashFile(ctx, src, srcPath)
	if err != nil {
		return false, err
	}
	b, err := hashFile(ctx, dst, dstPath)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func hashFile(ctx context.Context, fsys FileSystem, p string) ([32]byte, error) {
	r, err := fsys.OpenRead(ctx, p)
	if err != nil {
		return [32]byte{}, err
	}
	defer r.Close()
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, ctx.Err()
}

func sameModTime(a, b time.Time) bool {
	d := a.Sub(b)
	if d < 0 {
		d = -d
	}
	return d <= time.Second
}

func deleteExtraneous(ctx context.Context, dst FileSystem, dstRoot string, remote bool, sourceIndex map[string]FileInfo, excluder Excluder, opts Options) (int, error) {
	if _, err := dst.Stat(ctx, dstRoot); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	var entries []FileInfo
	err := dst.Walk(ctx, dstRoot, func(p string, info FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := relPath(remote, dstRoot, p)
		if err != nil {
			return err
		}
		rel = normalizeRel(rel)
		if rel == "." || excluder.Match(rel, info.IsDir) {
			return nil
		}
		if _, ok := sourceIndex[rel]; !ok {
			info.Path = p
			entries = append(entries, info)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return len(entries[i].Path) > len(entries[j].Path)
	})
	deleted := 0
	for _, entry := range entries {
		if opts.DryRun {
			deleted++
			opts.Logger.Info("delete", "path", entry.Path)
			continue
		}
		var err error
		if entry.IsDir {
			err = dst.RemoveAll(ctx, entry.Path)
		} else {
			err = dst.Remove(ctx, entry.Path)
		}
		if err != nil {
			return deleted, err
		}
		deleted++
		opts.Logger.Info("delete", "path", entry.Path)
	}
	return deleted, nil
}
