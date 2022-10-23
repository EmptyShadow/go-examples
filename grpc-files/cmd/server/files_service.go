package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
)

var ErrFileNotFound = errors.New("file not found")

type FilesSystem interface {
	SaveFile(ctx context.Context, name string, content io.Reader) (size uint64, err error)
	ReadFile(ctx context.Context, name string) (size uint64, content io.ReadCloser, err error)
}

type FilesService struct {
	filesSystem FilesSystem
}

func NewFilesService(filesSystem FilesSystem) *FilesService {
	return &FilesService{
		filesSystem: filesSystem,
	}
}

type FileInfo struct {
	Name string
}

type FileHeader struct {
	Name        string
	ContentType string
	Size        uint64
}

func (s *FilesService) UploadFile(ctx context.Context, info FileInfo, fileContent io.Reader) (*FileHeader, error) {
	extension := filepath.Ext(info.Name)

	size, err := s.filesSystem.SaveFile(ctx, info.Name, fileContent)
	if err != nil {
		return nil, fmt.Errorf("save file in file system: %w", err)
	}

	h := FileHeader{
		Name:        info.Name,
		ContentType: extension, // TODO: detect content type by extension.
		Size:        size,
	}

	return &h, nil
}

func (s *FilesService) DownloadFile(ctx context.Context, name string) (*FileHeader, io.ReadCloser, error) {
	size, fileContent, err := s.filesSystem.ReadFile(ctx, name)
	if err != nil {
		return nil, nil, fmt.Errorf("start read file: %w", err)
	}
	if fileContent == nil {
		return nil, nil, ErrFileNotFound
	}

	extension := filepath.Ext(name)

	return &FileHeader{
		Name:        name,
		ContentType: extension, // TODO: detect content type by extension.
		Size:        size,
	}, fileContent, nil
}
