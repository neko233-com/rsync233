package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
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
	fs := flag.NewFlagSet("rsync233", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts rsync233.Options
	var includes multiValue
	var excludes multiValue
	var checksum bool
	var quiet bool

	fs.BoolVar(&opts.Archive, "a", true, "archive mode: copy mode and modification time")
	fs.BoolVar(&opts.Archive, "archive", true, "archive mode: copy mode and modification time")
	fs.BoolVar(&opts.Recursive, "r", false, "recurse into directories")
	fs.BoolVar(&opts.Recursive, "recursive", false, "recurse into directories")
	fs.BoolVar(&opts.Links, "l", false, "copy symlinks as symlinks")
	fs.BoolVar(&opts.Links, "links", false, "copy symlinks as symlinks")
	fs.BoolVar(&opts.Delete, "delete", false, "delete destination files not present in source")
	fs.BoolVar(&opts.DryRun, "n", false, "show changes without writing")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "show changes without writing")
	fs.BoolVar(&opts.Check, "check", false, "exit 2 when destination differs")
	fs.BoolVar(&checksum, "c", false, "compare file checksums when size and time match")
	fs.BoolVar(&checksum, "checksum", false, "compare file checksums when size and time match")
	fs.BoolVar(&opts.IgnoreTimes, "I", false, "do not skip files that match size and modification time")
	fs.BoolVar(&opts.IgnoreTimes, "ignore-times", false, "do not skip files that match size and modification time")
	fs.BoolVar(&opts.SizeOnly, "size-only", false, "skip files that have matching size regardless of modification time")
	fs.BoolVar(&opts.IgnoreExisting, "ignore-existing", false, "skip updating files that already exist on the receiver")
	fs.BoolVar(&opts.Existing, "existing", false, "skip creating files that do not already exist on the receiver")
	fs.BoolVar(&opts.Update, "u", false, "skip files that are newer on the receiver")
	fs.BoolVar(&opts.Update, "update", false, "skip files that are newer on the receiver")
	fs.BoolVar(&opts.PreserveOwner, "owner", false, "reserved for platforms that support ownership preservation")
	fs.BoolVar(&opts.Progress, "progress", false, "reserved for progress output")
	fs.BoolVar(&quiet, "q", false, "suppress normal output")
	fs.BoolVar(&quiet, "quiet", false, "suppress normal output")
	fs.Var(&includes, "include", "include path pattern before exclude checks; may be repeated")
	fs.Var(&excludes, "exclude", "exclude path pattern; may be repeated")

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
	opts.Includes = includes
	opts.Excludes = excludes
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
