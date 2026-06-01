package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepo      = "neko233-com/rsync233"
	updateBinary    = "rsync233"
	githubAPI       = "https://api.github.com/repos/" + updateRepo + "/releases"
	updateUserAgent = "rsync233/" + updateBinary
)

type updateOptions struct {
	CheckOnly bool
	Yes       bool
	Version   string
}

func runUpdateCommand(args []string, stdout, stderr io.Writer) error {
	var opts updateOptions
	fs := flag.NewFlagSet("rsync233 update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&opts.CheckOnly, "check", false, "only check for updates, do not install")
	fs.BoolVar(&opts.Yes, "y", false, "install without confirmation")
	fs.BoolVar(&opts.Yes, "yes", false, "install without confirmation")
	fs.StringVar(&opts.Version, "version", "", "install a specific version, for example v1.0.0")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: rsync233 update [--check] [-y|--yes] [--version vX.Y.Z]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("unexpected update argument %q", fs.Arg(0))
	}
	return runUpdate(opts, stdout, os.Stdin)
}

func runUpdate(opts updateOptions, stdout io.Writer, stdin io.Reader) error {
	fmt.Fprintf(stdout, "Current version: %s\n", version)

	targetTag, err := resolveTargetVersion(opts.Version)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Latest version:  %s\n", targetTag)

	if opts.Version == "" && version != "dev" {
		if normalizeVersionTag(targetTag) == normalizeVersionTag(version) {
			fmt.Fprintln(stdout, "You are running the latest version.")
			return nil
		}
		if !versionLess(version, targetTag) {
			fmt.Fprintln(stdout, "You are running a newer build than the latest release.")
			return nil
		}
	}

	if opts.CheckOnly {
		fmt.Fprintln(stdout, "A newer version is available. Run: rsync233 update -y")
		return nil
	}

	if !opts.Yes {
		fmt.Fprint(stdout, "Install update? [y/N]: ")
		var answer string
		_, _ = fmt.Fscanln(stdin, &answer)
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(stdout, "Cancelled.")
			return nil
		}
	}

	execPath, err := resolveExecutable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	url := releaseAssetURL(targetTag, runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(stdout, "Downloading %s...\n", url)

	tmpPath, err := downloadReleaseBinary(url)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Installing to %s...\n", execPath)
	if err := applySelfUpdate(tmpPath, execPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	fmt.Fprintf(stdout, "Updated to %s\n", targetTag)
	return nil
}

func resolveTargetVersion(requested string) (string, error) {
	if requested != "" {
		return formatVersionTag(requested), nil
	}
	return fetchLatestReleaseTag()
}

func formatVersionTag(v string) string {
	v = strings.TrimSpace(v)
	for strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		v = v[1:]
	}
	if v == "" {
		return "v0.0.0"
	}
	return "v" + v
}

func normalizeVersionTag(v string) string {
	return strings.TrimPrefix(formatVersionTag(v), "v")
}

func fetchLatestReleaseTag() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, githubAPI+"/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", updateUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("GitHub API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse release: %w", err)
	}
	if result.TagName == "" {
		return "", errors.New("empty tag_name in latest release")
	}
	return result.TagName, nil
}

func releaseAssetURL(tag, goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s-%s-%s%s",
		updateRepo, formatVersionTag(tag), updateBinary, goos, goarch, ext,
	)
}

func downloadReleaseBinary(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", updateUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("download %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("rsync233-update-%d", os.Getpid()))
	if runtime.GOOS == "windows" {
		tmpPath += ".exe"
	}

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func resolveExecutable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}

func applySelfUpdate(src, dst string) error {
	if runtime.GOOS == "windows" {
		return applySelfUpdateWindows(src, dst)
	}
	return applySelfUpdateUnix(src, dst)
}

func applySelfUpdateUnix(src, dst string) error {
	mode := os.FileMode(0o755)
	if info, err := os.Stat(dst); err == nil {
		mode = info.Mode()
	}
	if err := os.Chmod(src, mode); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("replace binary; try again with elevated permissions: %w", err)
	}
	_ = os.Remove(src)
	return nil
}

func applySelfUpdateWindows(src, dst string) error {
	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("rsync233-update-%d.cmd", os.Getpid()))
	script := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /Y "%s" "%s" >nul
if errorlevel 1 exit /b 1
del "%%~f0"
`, src, dst)

	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		return err
	}

	c := exec.Command("cmd", "/c", "start", "", "/min", scriptPath)
	if err := c.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("start updater: %w", err)
	}

	fmt.Println("Update will finish after this process exits.")
	os.Exit(0)
	return nil
}

func versionLess(a, b string) bool {
	pa := strings.Split(normalizeVersionTag(a), ".")
	pb := strings.Split(normalizeVersionTag(b), ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(pa) {
			ai, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			bi, _ = strconv.Atoi(pb[i])
		}
		if ai != bi {
			return ai < bi
		}
	}
	return false
}
