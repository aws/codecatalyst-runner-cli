package fs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/format/index"
)

// FileCollectorFs provides an interface for a file system
type FileCollectorFs interface {
	Walk(root string, fn filepath.WalkFunc) error
	OpenGitIndex(path string) (*index.Index, error)
	Open(path string) (io.ReadCloser, error)
	Readlink(path string) (string, error)
}

// FileCollectorHandler provides an interface to collect files
type FileCollectorHandler interface {
	WriteFile(path string, fi fs.FileInfo, linkName string, f io.Reader) error
}

// FileCollector collects files from a git index
type FileCollector struct {
	Ignorer   gitignore.Matcher
	SrcPath   string
	SrcPrefix string
	Fs        FileCollectorFs
	Handler   FileCollectorHandler
}

// CollectFiles provides a WalkFunc to collect files in a git index
//
//nolint:gocyclo
func (fc *FileCollector) CollectFiles(ctx context.Context, submodulePath []string) filepath.WalkFunc {
	i, _ := fc.Fs.OpenGitIndex(path.Join(fc.SrcPath, path.Join(submodulePath...)))
	return func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("copy cancelled")
			default:
			}
		}

		sansPrefix := strings.TrimPrefix(file, fc.SrcPrefix)
		split := strings.Split(sansPrefix, string(filepath.Separator))
		// The root folders should be skipped, submodules only have the last path component set to "." by filepath.Walk
		if fi.IsDir() && len(split) > 0 && split[len(split)-1] == "." {
			return nil
		}
		var entry *index.Entry
		if i != nil {
			entry, err = i.Entry(strings.Join(split[len(submodulePath):], "/"))
		} else {
			err = index.ErrEntryNotFound
		}
		if err != nil && fc.Ignorer != nil && fc.Ignorer.Match(split, fi.IsDir()) {
			if fi.IsDir() {
				if i != nil {
					ms, err := i.Glob(strings.Join(append(split[len(submodulePath):], "**"), "/"))
					if err != nil || len(ms) == 0 {
						return filepath.SkipDir
					}
				} else {
					return filepath.SkipDir
				}
			} else {
				return nil
			}
		}
		if err == nil && entry.Mode == filemode.Submodule {
			err = fc.Fs.Walk(file, fc.CollectFiles(ctx, split))
			if err != nil {
				return err
			}
			return filepath.SkipDir
		}
		path := filepath.ToSlash(sansPrefix)

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			linkName, err := fc.Fs.Readlink(file)
			if err != nil {
				return fmt.Errorf("unable to readlink '%s': %w", file, err)
			}
			return fc.Handler.WriteFile(path, fi, linkName, nil)
		} else if !fi.Mode().IsRegular() {
			return nil
		}

		// open file
		f, err := fc.Fs.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		if ctx != nil {
			// make io.Copy cancellable by closing the file
			cpctx, cpfinish := context.WithCancel(ctx)
			defer cpfinish()
			go func() {
				select {
				case <-cpctx.Done():
				case <-ctx.Done():
					f.Close()
				}
			}()
		}

		return fc.Handler.WriteFile(path, fi, "", f)
	}
}
