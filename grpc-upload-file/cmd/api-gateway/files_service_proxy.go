package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/EmptyShadow/go-examples/grpc-upload-file/pb/pb/files/v1"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type FilesServiceProxy struct {
	filesServiceClient files.FilesServiceClient
	marshler           gwruntime.Marshaler
}

func NewFilesServiceProxy(
	filesServiceClient files.FilesServiceClient,
	marshler gwruntime.Marshaler,
) *FilesServiceProxy {
	return &FilesServiceProxy{
		filesServiceClient: filesServiceClient,
		marshler:           marshler,
	}
}

func (p *FilesServiceProxy) RegistrationHTTP(mux *gwruntime.ServeMux) {
	mux.HandlePath(http.MethodPost, "/v1/files/upload", p.UploadFile)
}

func (p *FilesServiceProxy) UploadFile(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err.Error()), http.StatusBadRequest)
		return
	}

	f, header, err := r.FormFile("attachment")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get file 'attachment': %s", err.Error()), http.StatusBadRequest)
		return
	}
	defer f.Close()

	stream, err := p.filesServiceClient.UploadFile(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed start stream to grpc files service: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	err = stream.Send(&files.UploadFileRequest{
		Data: &files.UploadFileRequest_FileInfo{
			FileInfo: &files.UploadFileInfo{
				Name: header.Filename,
			},
		},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed send upload file info to grpc files service: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	chunk := make([]byte, 1024)

	for {
		n, err := f.Read(chunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("failed read next file content chunk: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		err = stream.Send(&files.UploadFileRequest{
			Data: &files.UploadFileRequest_FileContentChunk{
				FileContentChunk: chunk[:n],
			},
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed send file content chunk to grpc files service: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalln(fmt.Errorf("failed close grpc stream and receive upload file response: %w", err))
		return
	}

	if err = p.marshler.NewEncoder(w).Encode(res); err != nil {
		log.Fatalln(fmt.Errorf("failed write response: %w", err))
		return
	}
}
