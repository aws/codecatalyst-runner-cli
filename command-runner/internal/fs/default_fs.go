package fs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/index"
)

// DefaultFs provides a default implementation of the [FileCollectorFS] interface
type DefaultFs struct {
}

// Walk walks the file tree rooted at root, calling fn for each file
func (*DefaultFs) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

// OpenGitIndex opens the git index file
func (*DefaultFs) OpenGitIndex(path string) (*index.Index, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	i, err := r.Storer.Index()
	if err != nil {
		return nil, err
	}
	return i, nil
}

// Open opens a path
func (*DefaultFs) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// Readlink reads a link at path
func (*DefaultFs) Readlink(path string) (string, error) {
	return os.Readlink(path)
}
