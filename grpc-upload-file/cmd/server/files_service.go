package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
)

type FilesSystem interface {
	SaveFile(ctx context.Context, name string, content io.Reader) (size uint64, err error)
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
	Name      string
	Extension string
	Size      uint64
}

func (s *FilesService) UploadFile(ctx context.Context, info FileInfo, fileContent io.Reader) (*FileHeader, error) {
	extension := filepath.Ext(info.Name)

	size, err := s.filesSystem.SaveFile(ctx, info.Name, fileContent)
	if err != nil {
		return nil, fmt.Errorf("save file in file system: %w", err)
	}

	h := FileHeader{
		Name:      info.Name,
		Extension: extension,
		Size:      size,
	}

	return &h, nil
}
