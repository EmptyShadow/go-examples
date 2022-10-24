package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EmptyShadow/go-examples/grpc-files/pb/files/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FilesServiceServer struct {
	files.UnimplementedFilesServiceServer

	service               *FilesService
	uploadFileBufferSize  int
	downloadFileChunkSize int
}

func NewFilesServiceServer(service *FilesService) *FilesServiceServer {
	return &FilesServiceServer{
		service:               service,
		uploadFileBufferSize:  1024,
		downloadFileChunkSize: 1024,
	}
}

func (s *FilesServiceServer) RegistrationGRPC(r grpc.ServiceRegistrar) {
	files.RegisterFilesServiceServer(r, s)
}

func (s *FilesServiceServer) ListFilesHeader(ctx context.Context, _ *files.ListFilesHeaderRequest) (resp *files.ListFilesHeaderResponse, err error) {
	fileHeaders, err := s.service.ListFilesHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("get list of files header")
	}

	items := make([]*files.FileHeader, len(fileHeaders))

	for i := range fileHeaders {
		items[i] = &files.FileHeader{
			Name:        fileHeaders[i].Name,
			ContentType: fileHeaders[i].ContentType,
			Size:        fileHeaders[i].Size,
		}
	}

	return &files.ListFilesHeaderResponse{
		Items: items,
	}, nil
}

func (s *FilesServiceServer) UploadFile(stream files.FilesService_UploadFileServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("read first message: %w", err)
	}

	fileInfo := msg.GetFileInfo()
	if fileInfo == nil {
		return status.Error(codes.Internal, "first message need have type of UploadFileRequest_Info")
	}

	fileReader := bufio.NewReaderSize(NewFileContentReader(stream), s.uploadFileBufferSize)

	fileHeader, err := s.service.UploadFile(stream.Context(), fileInfo.GetName(), fileReader)
	if err != nil {
		return fmt.Errorf("handle upload file: %w", err)
	}

	err = stream.SendAndClose(&files.UploadFileResponse{
		FileHeader: &files.FileHeader{
			Name:        fileHeader.Name,
			ContentType: fileHeader.ContentType,
			Size:        fileHeader.Size,
		},
	})
	if err != nil {
		return fmt.Errorf("send response and close stream: %w", err)
	}

	return nil
}

func (s *FilesServiceServer) DownloadFile(req *files.DownloadFileRequest, stream files.FilesService_DownloadFileServer) error {
	fileHeader, fileContent, err := s.service.DownloadFile(stream.Context(), req.Name)
	if errors.Is(err, ErrFileNotFound) {
		return status.Error(codes.NotFound, ErrFileNotFound.Error())
	}
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	err = stream.Send(&files.DownloadFileResponse{
		Data: &files.DownloadFileResponse_FileHeader{
			FileHeader: &files.FileHeader{
				Name:        fileHeader.Name,
				ContentType: fileHeader.ContentType,
				Size:        fileHeader.Size,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send file header to stream: %w", err)
	}

	chunk := make([]byte, s.downloadFileChunkSize)

	for {
		n, err := fileContent.Read(chunk[:])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read chunk of file content: %w", err)
		}

		err = stream.Send(&files.DownloadFileResponse{
			Data: &files.DownloadFileResponse_FileContentChunk{
				FileContentChunk: chunk[:n],
			},
		})
		if err != nil {
			return fmt.Errorf("send chunk of file content to stream: %w", err)
		}
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
