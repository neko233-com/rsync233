package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/neko233-com/rsync233/internal/rsync233"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		code := 1
		if errors.Is(err, rsync233.ErrDiffFound) {
			code = 2
		} else {
			fmt.Fprintf(os.Stderr, "rsync233: %v\n", err)
		}
		os.Exit(code)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-version":
			runVersion(stdout)
			return nil
		case "update":
			return runUpdateCommand(args[1:], stdout, stderr)
		}
	}

	fs := flag.NewFlagSet("rsync233", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts rsync233.Options
	var includes multiValue
	var excludes multiValue
	var includeFrom multiValue
	var excludeFrom multiValue
	var filters multiValue
	var minSize sizeValue
	var maxSize sizeValue
	var checksum bool
	var quiet bool

	fs.BoolVar(&opts.Archive, "a", false, "archive mode: recurse and copy mode, modification time, and symlinks")
	fs.BoolVar(&opts.Archive, "archive", false, "archive mode: recurse and copy mode, modification time, and symlinks")
	fs.BoolVar(&opts.Recursive, "r", false, "recurse into directories")
	fs.BoolVar(&opts.Recursive, "recursive", false, "recurse into directories")
	fs.BoolVar(&opts.Links, "l", false, "copy symlinks as symlinks")
	fs.BoolVar(&opts.Links, "links", false, "copy symlinks as symlinks")
	fs.BoolVar(&opts.PreservePerms, "p", false, "preserve permissions")
	fs.BoolVar(&opts.PreservePerms, "perms", false, "preserve permissions")
	fs.BoolVar(&opts.NoPerms, "no-perms", false, "do not preserve permissions, even in archive mode")
	fs.BoolVar(&opts.PreserveTimes, "t", false, "preserve modification times")
	fs.BoolVar(&opts.PreserveTimes, "times", false, "preserve modification times")
	fs.BoolVar(&opts.NoTimes, "no-times", false, "do not preserve modification times, even in archive mode")
	fs.BoolVar(&opts.Delete, "delete", false, "delete destination files not present in source")
	fs.BoolVar(&opts.DeleteExcluded, "delete-excluded", false, "also delete excluded destination files when --delete is enabled")
	fs.BoolVar(&opts.DryRun, "n", false, "show changes without writing")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "show changes without writing")
	fs.BoolVar(&opts.Check, "check", false, "exit 2 when destination differs")
	fs.BoolVar(&checksum, "c", false, "compare file checksums when size and time match")
	fs.BoolVar(&checksum, "checksum", false, "compare file checksums when size and time match")
	fs.BoolVar(&opts.IgnoreTimes, "I", false, "do not skip files that match size and modification time")
	fs.BoolVar(&opts.IgnoreTimes, "ignore-times", false, "do not skip files that match size and modification time")
	fs.BoolVar(&opts.SizeOnly, "size-only", false, "skip files that have matching size regardless of modification time")
	fs.BoolVar(&opts.IgnoreExisting, "ignore-existing", false, "skip updating files that already exist on the receiver")
	fs.BoolVar(&opts.IgnoreMissingArgs, "ignore-missing-args", false, "ignore missing source arguments without error")
	fs.BoolVar(&opts.Existing, "existing", false, "skip creating files that do not already exist on the receiver")
	fs.BoolVar(&opts.Update, "u", false, "skip files that are newer on the receiver")
	fs.BoolVar(&opts.Update, "update", false, "skip files that are newer on the receiver")
	fs.BoolVar(&opts.PreserveOwner, "owner", false, "reserved for platforms that support ownership preservation")
	fs.BoolVar(&opts.Progress, "progress", false, "reserved for progress output")
	fs.BoolVar(&quiet, "q", false, "suppress normal output")
	fs.BoolVar(&quiet, "quiet", false, "suppress normal output")
	fs.Var(&includes, "include", "include path pattern before exclude checks; may be repeated")
	fs.Var(&excludes, "exclude", "exclude path pattern; may be repeated")
	fs.Var(&includeFrom, "include-from", "read include patterns from file; may be repeated")
	fs.Var(&excludeFrom, "exclude-from", "read exclude patterns from file; may be repeated")
	fs.Var(&filters, "f", "rsync-style filter rule such as '+ *.go' or '- cache/'; may be repeated")
	fs.Var(&filters, "filter", "rsync-style filter rule such as '+ *.go' or '- cache/'; may be repeated")
	fs.Var(&minSize, "min-size", "skip files smaller than SIZE; supports K, M, G, T suffixes")
	fs.Var(&maxSize, "max-size", "skip files larger than SIZE; supports K, M, G, T suffixes")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: rsync233 [options] SOURCE DEST")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Endpoints can be local paths, ssh://[user@]host[:port]/path, or user@host:path.")
		fmt.Fprintln(stderr, "Like rsync, a source ending in a path separator copies the directory contents.")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("expected SOURCE and DEST")
	}

	opts.Checksum = checksum
	opts.MinSize = int64(minSize)
	opts.MaxSize = int64(maxSize)
	fileIncludes, err := readPatternFiles(includeFrom)
	if err != nil {
		return err
	}
	fileExcludes, err := readPatternFiles(excludeFrom)
	if err != nil {
		return err
	}
	filterRules, err := parseFilterRules(filters)
	if err != nil {
		return err
	}
	opts.Includes = append(includes, fileIncludes...)
	opts.Excludes = append(excludes, fileExcludes...)
	opts.FilterRules = filterRules
	if quiet {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	} else {
		opts.Logger = slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	summary, err := rsync233.Sync(context.Background(), fs.Arg(0), fs.Arg(1), opts)
	if err != nil {
		return err
	}
	if !quiet {
		fmt.Fprintf(stdout, "%s\n", summary)
	}
	if opts.Check && summary.Changed() {
		return rsync233.ErrDiffFound
	}
	return nil
}

type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func readPatternFiles(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read pattern file %s: %w", p, err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, line)
		}
	}
	return out, nil
}

func parseFilterRules(values []string) ([]rsync233.FilterRule, error) {
	rules := make([]rsync233.FilterRule, 0, len(values))
	for _, raw := range values {
		include, pattern, err := parseFilterRule(raw)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rsync233.FilterRule{Include: include, Pattern: pattern})
	}
	return rules, nil
}

func parseFilterRule(raw string) (bool, string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return false, "", fmt.Errorf("empty filter rule")
	}
	for _, prefix := range []string{"+ ", "include "} {
		if strings.HasPrefix(v, prefix) {
			p := strings.TrimSpace(strings.TrimPrefix(v, prefix))
			if p == "" {
				return false, "", fmt.Errorf("empty include filter pattern")
			}
			return true, p, nil
		}
	}
	for _, prefix := range []string{"- ", "exclude "} {
		if strings.HasPrefix(v, prefix) {
			p := strings.TrimSpace(strings.TrimPrefix(v, prefix))
			if p == "" {
				return false, "", fmt.Errorf("empty exclude filter pattern")
			}
			return false, p, nil
		}
	}
	return false, "", fmt.Errorf("unsupported filter rule %q; use '+ pattern' or '- pattern'", raw)
}

type sizeValue int64

func (s *sizeValue) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

func (s *sizeValue) Set(v string) error {
	n, err := parseSize(v)
	if err != nil {
		return err
	}
	*s = sizeValue(n)
	return nil
}

func parseSize(v string) (int64, error) {
	raw := strings.TrimSpace(v)
	if raw == "" {
		return 0, fmt.Errorf("empty size")
	}
	multiplier := int64(1)
	last := raw[len(raw)-1]
	if last == 'b' || last == 'B' {
		raw = strings.TrimSpace(raw[:len(raw)-1])
		if raw == "" {
			return 0, fmt.Errorf("invalid size %q", v)
		}
		last = raw[len(raw)-1]
	}
	switch last {
	case 'k', 'K':
		multiplier = 1024
		raw = strings.TrimSpace(raw[:len(raw)-1])
	case 'm', 'M':
		multiplier = 1024 * 1024
		raw = strings.TrimSpace(raw[:len(raw)-1])
	case 'g', 'G':
		multiplier = 1024 * 1024 * 1024
		raw = strings.TrimSpace(raw[:len(raw)-1])
	case 't', 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		raw = strings.TrimSpace(raw[:len(raw)-1])
	}
	if raw == "" {
		return 0, fmt.Errorf("invalid size %q", v)
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", v)
	}
	if n > (1<<63-1)/multiplier {
		return 0, fmt.Errorf("size %q overflows int64", v)
	}
	return n * multiplier, nil
}
