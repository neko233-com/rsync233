package rsync233

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Endpoint struct {
	Raw       string
	Scheme    string
	User      string
	Host      string
	Port      int
	Path      string
	Trailing  bool
	LocalPath string
}

func ParseEndpoint(raw string) (Endpoint, error) {
	if raw == "" {
		return Endpoint{}, fmt.Errorf("empty endpoint")
	}
	ep := Endpoint{Raw: raw, Port: 22, Trailing: hasTrailingSeparator(raw)}

	if strings.HasPrefix(raw, "ssh://") || strings.HasPrefix(raw, "sftp://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Endpoint{}, err
		}
		if u.Hostname() == "" {
			return Endpoint{}, fmt.Errorf("remote endpoint %q is missing host", raw)
		}
		ep.Scheme = "ssh"
		ep.Host = u.Hostname()
		if u.User != nil {
			ep.User = u.User.Username()
		}
		if u.Port() != "" {
			port, err := strconv.Atoi(u.Port())
			if err != nil {
				return Endpoint{}, fmt.Errorf("invalid ssh port %q", u.Port())
			}
			ep.Port = port
		}
		ep.Path = u.Path
		if runtime.GOOS == "windows" && strings.HasPrefix(ep.Path, "/") && len(ep.Path) >= 3 && ep.Path[2] == ':' {
			ep.Path = ep.Path[1:]
		}
		if ep.Path == "" {
			ep.Path = "."
		}
		return ep, nil
	}

	if legacy, ok := parseScpLike(raw); ok {
		return legacy, nil
	}

	ep.Scheme = "file"
	ep.LocalPath = filepath.Clean(raw)
	ep.Path = ep.LocalPath
	return ep, nil
}

func (e Endpoint) IsRemote() bool {
	return e.Scheme == "ssh"
}

func (e Endpoint) Address() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

func parseScpLike(raw string) (Endpoint, bool) {
	if isWindowsDrivePath(raw) {
		return Endpoint{}, false
	}
	colon := strings.IndexByte(raw, ':')
	if colon <= 0 || strings.ContainsAny(raw[:colon], `/\`) {
		return Endpoint{}, false
	}
	left := raw[:colon]
	path := raw[colon+1:]
	user := ""
	host := left
	if at := strings.LastIndexByte(left, '@'); at >= 0 {
		user = left[:at]
		host = left[at+1:]
	}
	if host == "" {
		return Endpoint{}, false
	}
	return Endpoint{
		Raw:      raw,
		Scheme:   "ssh",
		User:     user,
		Host:     host,
		Port:     22,
		Path:     path,
		Trailing: hasTrailingSeparator(path),
	}, true
}

func hasTrailingSeparator(s string) bool {
	if s == "" {
		return false
	}
	clean := strings.TrimRight(s, " \t")
	return strings.HasSuffix(clean, "/") || strings.HasSuffix(clean, "\\") || clean == "."+string(os.PathSeparator)
}

func isWindowsDrivePath(s string) bool {
	if len(s) < 2 || s[1] != ':' {
		return false
	}
	drive := s[0]
	return (drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')
}
