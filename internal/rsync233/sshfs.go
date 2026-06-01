package rsync233

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SFTPFS struct {
	ssh  *ssh.Client
	sftp *sftp.Client
}

func OpenFS(ctx context.Context, ep Endpoint) (FileSystem, error) {
	if !ep.IsRemote() {
		return LocalFS{}, nil
	}
	return openSFTP(ctx, ep)
}

func openSFTP(ctx context.Context, ep Endpoint) (*SFTPFS, error) {
	cfg, err := sshConfig(ep.User)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 20 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", ep.Address())
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, ep.Address(), cfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	sshClient := ssh.NewClient(c, chans, reqs)
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	return &SFTPFS{ssh: sshClient, sftp: sftpClient}, nil
}

func sshConfig(name string) (*ssh.ClientConfig, error) {
	if name == "" {
		current, err := user.Current()
		if err == nil {
			name = current.Username
		}
	}
	auths, err := publicKeyAuths()
	if err != nil {
		return nil, err
	}
	hostKeyCallback, err := knownHostCallback()
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            name,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         20 * time.Second,
	}, nil
}

func publicKeyAuths() ([]ssh.AuthMethod, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	names := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	var methods []ssh.AuthMethod
	for _, name := range names {
		keyPath := filepath.Join(home, ".ssh", name)
		key, err := os.ReadFile(keyPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			continue
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("no usable SSH private key found in ~/.ssh")
	}
	return methods, nil
}

func knownHostCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHosts); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("known_hosts not found: %s", knownHosts)
	}
	return knownhosts.New(knownHosts)
}

func (s *SFTPFS) Stat(_ context.Context, p string) (FileInfo, error) {
	st, err := s.sftp.Lstat(p)
	if err != nil {
		return FileInfo{}, normalizeSFTPErr(err)
	}
	return FileInfo{Path: p, Mode: st.Mode(), Size: st.Size(), ModTime: st.ModTime(), IsDir: st.IsDir(), IsSymlink: st.Mode()&fs.ModeSymlink != 0}, nil
}

func (s *SFTPFS) MkdirAll(_ context.Context, p string, mode fs.FileMode) error {
	if err := s.sftp.MkdirAll(p); err != nil {
		return err
	}
	return s.sftp.Chmod(p, mode)
}

func (s *SFTPFS) OpenRead(_ context.Context, p string) (io.ReadCloser, error) {
	return s.sftp.Open(p)
}

func (s *SFTPFS) OpenWrite(_ context.Context, p string, mode fs.FileMode) (io.WriteCloser, error) {
	if err := s.sftp.MkdirAll(path.Dir(p)); err != nil {
		return nil, err
	}
	f, err := s.sftp.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return nil, err
	}
	_ = s.sftp.Chmod(p, mode)
	return f, nil
}

func (s *SFTPFS) ReadLink(_ context.Context, p string) (string, error) {
	return s.sftp.ReadLink(p)
}

func (s *SFTPFS) Symlink(_ context.Context, target, p string) error {
	if err := s.sftp.MkdirAll(path.Dir(p)); err != nil {
		return err
	}
	return s.sftp.Symlink(target, p)
}

func (s *SFTPFS) Remove(_ context.Context, p string) error {
	return s.sftp.Remove(p)
}

func (s *SFTPFS) RemoveAll(_ context.Context, p string) error {
	return s.sftp.RemoveAll(p)
}

func (s *SFTPFS) Chtimes(_ context.Context, p string, modTime time.Time) error {
	return s.sftp.Chtimes(p, modTime, modTime)
}

func (s *SFTPFS) Chmod(_ context.Context, p string, mode fs.FileMode) error {
	return s.sftp.Chmod(p, mode)
}

func (s *SFTPFS) Walk(ctx context.Context, root string, fn WalkFunc) error {
	w := s.sftp.Walk(root)
	for w.Step() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.Err(); err != nil {
			if callErr := fn(w.Path(), FileInfo{}, normalizeSFTPErr(err)); callErr != nil {
				return callErr
			}
			continue
		}
		st := w.Stat()
		if err := fn(w.Path(), FileInfo{Path: w.Path(), Mode: st.Mode(), Size: st.Size(), ModTime: st.ModTime(), IsDir: st.IsDir(), IsSymlink: st.Mode()&fs.ModeSymlink != 0}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *SFTPFS) Close() error {
	var err error
	if s.sftp != nil {
		err = s.sftp.Close()
	}
	if s.ssh != nil {
		if closeErr := s.ssh.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func normalizeSFTPErr(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return os.ErrNotExist
	}
	if strings := err.Error(); strings == "file does not exist" || strings == "no such file" {
		return os.ErrNotExist
	}
	return err
}
