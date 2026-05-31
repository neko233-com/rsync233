package rsync233

import (
	"context"
	"io"
	"io/fs"
	"time"
)

type FileSystem interface {
	Stat(ctx context.Context, path string) (FileInfo, error)
	MkdirAll(ctx context.Context, path string, mode fs.FileMode) error
	OpenRead(ctx context.Context, path string) (io.ReadCloser, error)
	OpenWrite(ctx context.Context, path string, mode fs.FileMode) (io.WriteCloser, error)
	Remove(ctx context.Context, path string) error
	RemoveAll(ctx context.Context, path string) error
	Chtimes(ctx context.Context, path string, modTime time.Time) error
	Chmod(ctx context.Context, path string, mode fs.FileMode) error
	Walk(ctx context.Context, root string, fn WalkFunc) error
	Close() error
}

type FileInfo struct {
	Path    string
	Mode    fs.FileMode
	Size    int64
	ModTime time.Time
	IsDir   bool
}

type WalkFunc func(path string, info FileInfo, err error) error
