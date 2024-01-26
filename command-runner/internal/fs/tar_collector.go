package fs

import (
	"archive/tar"
	"io"
	"io/fs"
	"path"

	"github.com/rs/zerolog/log"
)

// TarCollector collects files to a tar file
type TarCollector struct {
	TarWriter *tar.Writer // the writer to use for files
	UID       int         // UID to apply to files in the tar
	GID       int         // GID to apply to files in the tar
	DstDir    string      // DstDir is prefixed on files collected before being added to the tar file
}

// WriteFile adds a file at fpath to the tar file
func (tc TarCollector) WriteFile(fpath string, fi fs.FileInfo, linkName string, f io.Reader) error {
	// create a new dir/file header
	header, err := tar.FileInfoHeader(fi, linkName)
	if err != nil {
		return err
	}
	log.Trace().Msgf("Tarring %s", fpath)

	// update the name to correctly reflect the desired destination when untaring
	header.Name = path.Join(tc.DstDir, fpath)
	header.Mode = int64(fi.Mode())
	header.ModTime = fi.ModTime()
	header.Uid = tc.UID
	header.Gid = tc.GID

	// write the header
	if err := tc.TarWriter.WriteHeader(header); err != nil {
		return err
	}

	// this is a symlink no reader provided
	if f == nil {
		return nil
	}

	// copy file data into tar writer
	if _, err := io.Copy(tc.TarWriter, f); err != nil {
		return err
	}
	return nil
}
