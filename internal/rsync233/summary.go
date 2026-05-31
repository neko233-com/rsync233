package rsync233

import "fmt"

type Summary struct {
	ScannedDirs    int
	ScannedFiles   int
	CreatedDirs    int
	CopiedFiles    int
	UpdatedFiles   int
	DeletedEntries int
	SkippedEntries int
	BytesCopied    int64
	DryRun         bool
	CheckOnly      bool
}

func (s Summary) Changed() bool {
	return s.CreatedDirs+s.CopiedFiles+s.UpdatedFiles+s.DeletedEntries > 0
}

func (s Summary) String() string {
	mode := "applied"
	if s.DryRun {
		mode = "dry-run"
	}
	if s.CheckOnly {
		mode = "check"
	}
	return fmt.Sprintf("%s: scanned_dirs=%d scanned_files=%d created_dirs=%d copied=%d updated=%d deleted=%d skipped=%d bytes=%d",
		mode, s.ScannedDirs, s.ScannedFiles, s.CreatedDirs, s.CopiedFiles, s.UpdatedFiles, s.DeletedEntries, s.SkippedEntries, s.BytesCopied)
}
