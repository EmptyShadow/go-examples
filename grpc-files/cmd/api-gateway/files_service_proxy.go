package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/EmptyShadow/go-examples/grpc-files/pb/files/v1"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
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
	mux.HandlePath(http.MethodPost, "/v1/files", p.UploadFile)
	mux.HandlePath(http.MethodGet, "/v1/files/{name}", p.DownloadFile)
}

func (p *FilesServiceProxy) UploadFile(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("to parse form: %s", err.Error()), http.StatusBadRequest)
		return
	}

	f, header, err := r.FormFile("attachment")
	if err != nil {
		http.Error(w, fmt.Sprintf("to get file 'attachment': %s", err.Error()), http.StatusBadRequest)
		return
	}
	defer f.Close()

	stream, err := p.filesServiceClient.UploadFile(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("start stream to grpc files service: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	err = stream.Send(&files.UploadFileRequest{
		Data: &files.UploadFileRequest_FileInfo{
			FileInfo: &files.UploadFileRequest_Info{
				Name: header.Filename,
			},
		},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("send upload file info to grpc files service: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	chunk := make([]byte, 1024)

	for {
		n, err := f.Read(chunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("read next file content chunk: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		err = stream.Send(&files.UploadFileRequest{
			Data: &files.UploadFileRequest_FileContentChunk{
				FileContentChunk: chunk[:n],
			},
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("send file content chunk to grpc files service: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalln(fmt.Errorf("close grpc stream and receive upload file response: %w", err))
		return
	}

	if err = p.marshler.NewEncoder(w).Encode(res); err != nil {
		log.Fatalln(fmt.Errorf("write response: %w", err))
		return
	}
}

func (p *FilesServiceProxy) DownloadFile(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	var metadata gwruntime.ServerMetadata

	stream, err := p.filesServiceClient.DownloadFile(r.Context(), &files.DownloadFileRequest{
		Name: pathParams["name"],
	}, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	if err != nil {
		http.Error(w, fmt.Sprintf("start download file stream from grpc files service %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer stream.CloseSend()

	fileInfoMessage, err := stream.Recv()
	if err != nil {
		http.Error(w, fmt.Sprintf("recive msg with file info from grpc files service %s", err.Error()), http.StatusInternalServerError)
		return
	}

	fileHeader := fileInfoMessage.GetFileHeader()
	responseHeaders := w.Header()
	responseHeaders.Add("content-type", fileHeader.GetExtension())
	responseHeaders.Add("content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileHeader.GetName()))

	for {
		chunkMessage, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("recive msg with file content chunk from grpc files service %s", err.Error()), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(chunkMessage.GetFileContentChunk())
		if err != nil {
			http.Error(w, fmt.Sprintf("write file content chunk to response %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}
}
