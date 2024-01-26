package fs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyCollector collects files by copying them to a destination directory
type CopyCollector struct {
	DstDir string // the destination directory to copy to
}

// WriteFile writes the given fpath to the destination directory for this [CopyCollector]
func (cc *CopyCollector) WriteFile(fpath string, fi fs.FileInfo, linkName string, f io.Reader) error {
	fdestpath := filepath.Join(cc.DstDir, fpath)
	if err := os.MkdirAll(filepath.Dir(fdestpath), 0o755); err != nil {
		return err
	}
	if f == nil {
		return os.Symlink(linkName, fdestpath)
	}
	df, err := os.OpenFile(fdestpath, os.O_CREATE|os.O_WRONLY, fi.Mode())
	if err != nil {
		return err
	}
	defer df.Close()
	if _, err := io.Copy(df, f); err != nil {
		return err
	}
	return nil
}
