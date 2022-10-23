package main

import (
	"context"
	"errors"
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

func (s *LocalFileSystem) ReadFile(ctx context.Context, name string) (size uint64, content io.ReadCloser, err error) {
	name = filepath.Join(s.root, name)

	f, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, fmt.Errorf("open file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return 0, nil, fmt.Errorf("get file info: %w", err)
	}

	return uint64(info.Size()), f, nil
}
