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
	Archive        bool
	Recursive      bool
	Delete         bool
	DeleteExcluded bool
	DryRun         bool
	Check          bool
	Checksum       bool
	Links          bool
	PreservePerms  bool
	NoPerms        bool
	IgnoreTimes    bool
	SizeOnly       bool
	IgnoreExisting bool
	Existing       bool
	Update         bool
	PreserveOwner  bool
	Progress       bool
	Includes       []string
	Excludes       []string
	Logger         *slog.Logger
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
	if srcInfo.IsDir && !opts.Archive && !opts.Recursive {
		return summary, fmt.Errorf("skipping directory %s; use -r or -a to recurse", sourceRaw)
	}
	if srcInfo.IsDir && !srcEP.Trailing {
		dstRoot = joinPath(dstEP.IsRemote(), dstRoot, endpointBase(srcRoot))
	}

	filter := NewFilter(opts.Includes, opts.Excludes)
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
		if filter.Exclude(rel, info.IsDir) {
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
		if info.IsSymlink {
			copied, updated, err := syncSymlink(ctx, srcFS, dstFS, srcPath, dstPath, opts)
			if err != nil {
				return err
			}
			if copied {
				summary.CopiedFiles++
				opts.Logger.Info("symlink", "path", dstPath)
			}
			if updated {
				summary.UpdatedFiles++
				opts.Logger.Info("symlink-update", "path", dstPath)
			}
			return nil
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
		deleted, err := deleteExtraneous(ctx, dstFS, dstRoot, dstEP.IsRemote(), sourceIndex, filter, opts)
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
		if !infoIsDir(ctx, dst, dstPath) {
			return false, fmt.Errorf("destination %s exists and is not a directory", dstPath)
		}
		if preservePerms(opts) && !opts.DryRun {
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
	return true, dst.MkdirAll(ctx, dstPath, dirCreateMode(mode, opts))
}

func infoIsDir(ctx context.Context, fsys FileSystem, p string) bool {
	info, err := fsys.Stat(ctx, p)
	return err == nil && info.IsDir
}

func syncFile(ctx context.Context, src, dst FileSystem, srcPath, dstPath string, srcInfo FileInfo, opts Options) (copied bool, updated bool, bytesCopied int64, err error) {
	dstInfo, err := dst.Stat(ctx, dstPath)
	if err == nil && dstInfo.IsDir {
		return false, false, 0, fmt.Errorf("destination %s is a directory", dstPath)
	}
	needsCopy := false
	isCreate := false
	if errors.Is(err, os.ErrNotExist) {
		if opts.Existing {
			return false, false, 0, nil
		}
		needsCopy = true
		isCreate = true
	} else if err != nil {
		return false, false, 0, err
	} else if opts.IgnoreExisting {
		return false, false, 0, nil
	} else if opts.Update && dstInfo.ModTime.After(srcInfo.ModTime) {
		return false, false, 0, nil
	} else if opts.IgnoreTimes {
		needsCopy = true
	} else if opts.SizeOnly {
		needsCopy = dstInfo.Size != srcInfo.Size
	} else if opts.Checksum {
		if dstInfo.Size != srcInfo.Size {
			needsCopy = true
		} else {
			equal, err := sameChecksum(ctx, src, dst, srcPath, dstPath)
			if err != nil {
				return false, false, 0, err
			}
			needsCopy = !equal
		}
	} else if dstInfo.Size != srcInfo.Size || !sameModTime(dstInfo.ModTime, srcInfo.ModTime) {
		needsCopy = true
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
	if !isCreate && dstInfo.IsSymlink {
		if err := dst.Remove(ctx, dstPath); err != nil {
			return false, false, 0, err
		}
	}
	if err := copyFile(ctx, src, dst, srcPath, dstPath, fileCreateMode(srcInfo.Mode, opts)); err != nil {
		return false, false, 0, err
	}
	if opts.Archive {
		_ = dst.Chtimes(ctx, dstPath, srcInfo.ModTime)
	}
	if preservePerms(opts) {
		_ = dst.Chmod(ctx, dstPath, srcInfo.Mode.Perm())
	}
	return isCreate, !isCreate, srcInfo.Size, nil
}

func syncSymlink(ctx context.Context, src, dst FileSystem, srcPath, dstPath string, opts Options) (copied bool, updated bool, err error) {
	if !opts.Archive && !opts.Links {
		return false, false, nil
	}
	target, err := src.ReadLink(ctx, srcPath)
	if err != nil {
		return false, false, err
	}
	dstInfo, statErr := dst.Stat(ctx, dstPath)
	isCreate := false
	if errors.Is(statErr, os.ErrNotExist) {
		isCreate = true
	} else if statErr != nil {
		return false, false, statErr
	} else if dstInfo.IsSymlink {
		current, err := dst.ReadLink(ctx, dstPath)
		if err != nil {
			return false, false, err
		}
		if current == target {
			return false, false, nil
		}
	} else if dstInfo.IsDir {
		return false, false, fmt.Errorf("destination %s is a directory", dstPath)
	}
	if opts.Check || opts.DryRun {
		return isCreate, !isCreate, nil
	}
	if !isCreate {
		if err := dst.Remove(ctx, dstPath); err != nil {
			return false, false, err
		}
	}
	if err := dst.Symlink(ctx, target, dstPath); err != nil {
		return false, false, err
	}
	return isCreate, !isCreate, nil
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

func preservePerms(opts Options) bool {
	return (opts.Archive || opts.PreservePerms) && !opts.NoPerms
}

func fileCreateMode(mode fs.FileMode, opts Options) fs.FileMode {
	if preservePerms(opts) {
		return mode.Perm()
	}
	return 0o666
}

func dirCreateMode(mode fs.FileMode, opts Options) fs.FileMode {
	if preservePerms(opts) {
		return mode.Perm()
	}
	return 0o777
}

func deleteExtraneous(ctx context.Context, dst FileSystem, dstRoot string, remote bool, sourceIndex map[string]FileInfo, filter Filter, opts Options) (int, error) {
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
		if rel == "." {
			return nil
		}
		if filter.Exclude(rel, info.IsDir) && !opts.DeleteExcluded {
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
