package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/EmptyShadow/go-examples/grpc-files/pb/files/v1"
	"google.golang.org/grpc"
)

type FilesServiceServer struct {
	files.UnimplementedFilesServiceServer

	service *FilesService
	bufsize int
}

func NewFilesServiceServer(service *FilesService) *FilesServiceServer {
	return &FilesServiceServer{
		service: service,
		bufsize: 4096,
	}
}

func (s *FilesServiceServer) RegistrationGRPC(r grpc.ServiceRegistrar) {
	files.RegisterFilesServiceServer(r, s)
}

func (s *FilesServiceServer) UploadFile(stream files.FilesService_UploadFileServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("read first message: %w", err)
	}

	fileInfo := FileInfo{
		Name: msg.GetFileInfo().GetName(),
	}

	fileReader := bufio.NewReaderSize(NewFileContentReader(stream), s.bufsize)

	fileHeader, err := s.service.UploadFile(stream.Context(), fileInfo, fileReader)
	if err != nil {
		return fmt.Errorf("handle upload file: %w", err)
	}

	err = stream.SendAndClose(&files.UploadFileResponse{
		FileHeader: &files.FileHeader{
			Name:      fileHeader.Name,
			Extension: fileHeader.Extension,
			Size:      fileHeader.Size,
		},
	})
	if err != nil {
		return fmt.Errorf("send response and close stream: %w", err)
	}

	return nil
}

type FileContentReader struct {
	stream files.FilesService_UploadFileServer
}

func NewFileContentReader(stream files.FilesService_UploadFileServer) *FileContentReader {
	return &FileContentReader{
		stream: stream,
	}
}

func (r *FileContentReader) Read(dst []byte) (int, error) {
	msg, err := r.stream.Recv()
	if errors.Is(err, io.EOF) {
		return 0, err
	}
	if err != nil {
		return 0, fmt.Errorf("read next file chunk: %w", err)
	}

	chunk := msg.GetFileContentChunk()
	n := copy(dst, chunk)

	return n, nil
}
