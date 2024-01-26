package fs

import (
	"archive/zip"
	"io"
	"io/fs"
)

// ZipCollector is a Collector that writes files to a zip file
type ZipCollector struct {
	ZipWriter *zip.Writer // the zip writer to write files to
}

// WriteFile writes files to a zip file
func (zc ZipCollector) WriteFile(fpath string, fi fs.FileInfo, _ string, f io.Reader) error {
	// create a new dir/file header
	header, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}

	header.Name = fpath
	header.SetMode(fi.Mode())
	header.Modified = fi.ModTime()

	// write the header
	writer, err := zc.ZipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// copy file data into zip writer
	if _, err := io.Copy(writer, f); err != nil {
		return err
	}
	return nil
}
