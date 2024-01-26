package shared

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"

	"github.com/go-git/go-billy/v5/helper/polyfill"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/rs/zerolog/log"
)

func TarDirectory(ctx context.Context, srcPath string, dstDir string, useGitIgnore bool, uid int, gid int) (*os.File, error) {
	tarFile, err := os.CreateTemp(fs.TmpDir(), "tardir")
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(tarFile)

	srcPrefix := filepath.Dir(srcPath)
	if !strings.HasSuffix(srcPrefix, string(filepath.Separator)) {
		srcPrefix += string(filepath.Separator)
	}
	log.Ctx(ctx).Printf("Stripping prefix:%s src:%s", srcPrefix, srcPath)

	var ignorer gitignore.Matcher
	if useGitIgnore {
		ps, err := gitignore.ReadPatterns(polyfill.New(osfs.New(srcPath)), nil)
		if err != nil {
			log.Ctx(ctx).Printf("Error loading .gitignore: %v", err)
		}

		ignorer = gitignore.NewMatcher(ps)
	}

	fc := &fs.FileCollector{
		Fs:        &fs.DefaultFs{},
		Ignorer:   ignorer,
		SrcPath:   srcPath,
		SrcPrefix: srcPrefix,
		Handler: &fs.TarCollector{
			TarWriter: tw,
			UID:       uid,
			GID:       gid,
			DstDir:    dstDir,
		},
	}

	err = filepath.Walk(srcPath, fc.CollectFiles(ctx, []string{}))
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	_, err = tarFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek tar archive: %w", err)
	}
	return tarFile, nil
}

func TarFiles(ctx context.Context, out io.Writer, uid int, gid int, files ...*types.FileEntry) error {
	tw := tar.NewWriter(out)
	for _, file := range files {
		log.Ctx(ctx).Printf("Writing entry to tarball %s len:%d", file.Name, len(file.Body))
		hdr := &tar.Header{
			Name: file.Name,
			Mode: file.Mode,
			Size: int64(len(file.Body)),
			Uid:  uid,
			Gid:  gid,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			return err
		}
	}
	return tw.Close()
}
