package rsync233

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type LocalFS struct{}

func (LocalFS) Stat(_ context.Context, path string) (FileInfo, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromOS(path, st), nil
}

func (LocalFS) MkdirAll(_ context.Context, path string, mode fs.FileMode) error {
	return os.MkdirAll(path, mode)
}

func (LocalFS) OpenRead(_ context.Context, path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (LocalFS) OpenWrite(_ context.Context, path string, mode fs.FileMode) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
}

func (LocalFS) ReadLink(_ context.Context, path string) (string, error) {
	return os.Readlink(path)
}

func (LocalFS) Symlink(_ context.Context, target, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.Symlink(target, path)
}

func (LocalFS) Rename(_ context.Context, oldPath, newPath string) error {
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(newPath)
	return os.Rename(oldPath, newPath)
}

func (LocalFS) Remove(_ context.Context, path string) error {
	return os.Remove(path)
}

func (LocalFS) RemoveAll(_ context.Context, path string) error {
	return os.RemoveAll(path)
}

func (LocalFS) Chtimes(_ context.Context, path string, modTime time.Time) error {
	return os.Chtimes(path, modTime, modTime)
}

func (LocalFS) Chmod(_ context.Context, path string, mode fs.FileMode) error {
	return os.Chmod(path, mode)
}

func (LocalFS) Walk(ctx context.Context, root string, fn WalkFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fn(path, FileInfo{}, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		st, err := d.Info()
		if err != nil {
			return fn(path, FileInfo{}, err)
		}
		return fn(path, fileInfoFromOS(path, st), nil)
	})
}

func (LocalFS) Close() error { return nil }

func fileInfoFromOS(path string, st os.FileInfo) FileInfo {
	return FileInfo{
		Path:      path,
		Mode:      st.Mode(),
		Size:      st.Size(),
		ModTime:   st.ModTime(),
		IsDir:     st.IsDir(),
		IsSymlink: st.Mode()&fs.ModeSymlink != 0,
	}
}
