package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var _ FilesSystem = (*LocalFileSystem)(nil)

type LocalFileSystem struct {
	root string
}

func MustNewLocalFileSystem(root string) *LocalFileSystem {
	lfs, err := NewLocalFileSystem(root)
	if err != nil {
		panic(fmt.Errorf("fatal init local file system: %w", err))
	}
	return lfs
}

func NewLocalFileSystem(root string) (_ *LocalFileSystem, err error) {
	if root == "" {
		root, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home dir path from os: %w", err)
		}
	}

	lsf := LocalFileSystem{
		root: root,
	}

	return &lsf, nil
}

func (s *LocalFileSystem) SaveFile(ctx context.Context, name string, content io.Reader) (size uint64, err error) {
	name = filepath.Join(s.root, name)

	f, err := os.Create(name)
	if err != nil {
		return 0, fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, content)
	if err != nil {
		return 0, fmt.Errorf("copy passed content to new file: %w", err)
	}

	return uint64(written), nil
}
