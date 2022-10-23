package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/EmptyShadow/go-examples/grpc-files/pb/files/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FilesServiceProxy struct {
	filesServiceClient files.FilesServiceClient
	mux                *gwruntime.ServeMux

	uploadFileChunkSize int
}

func NewFilesServiceProxy(
	filesServiceClient files.FilesServiceClient,
	mux *gwruntime.ServeMux,
) *FilesServiceProxy {
	return &FilesServiceProxy{
		filesServiceClient:  filesServiceClient,
		mux:                 mux,
		uploadFileChunkSize: 1024,
	}
}

func (p *FilesServiceProxy) RegistrationHTTP(mux *gwruntime.ServeMux) {
	mux.HandlePath(http.MethodPost, uploadFilePathPattern, p.UploadFile)
	mux.HandlePath(http.MethodGet, downloadFilePathPattern, p.DownloadFile)
}

const uploadFilePathPattern = "/v1/files"

func (p *FilesServiceProxy) UploadFile(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	_, outboundMarshaler := runtime.MarshalerForRequest(p.mux, req)

	ctx, err := runtime.AnnotateContext(req.Context(), p.mux, req, "/example.files.v1.FilesService/UploadFile", runtime.WithHTTPPathPattern(uploadFilePathPattern))
	if err != nil {
		runtime.HTTPError(ctx, p.mux, outboundMarshaler, w, req, err)
		return
	}

	res, err := p.uploadFile(ctx, req)
	if err != nil {
		runtime.HTTPError(ctx, p.mux, outboundMarshaler, w, req, err)
		return
	}

	gwruntime.ForwardResponseMessage(ctx, p.mux, outboundMarshaler, w, req, res, p.mux.GetForwardResponseOptions()...)
}

const formFileName = "attachment"

func (p *FilesServiceProxy) uploadFile(ctx context.Context, req *http.Request) (resp *files.UploadFileResponse, err error) {
	if err = req.ParseForm(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form: %s", err)
	}

	f, header, err := req.FormFile(formFileName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form file %s: %s", formFileName, err)
	}
	defer f.Close()

	stream, err := p.filesServiceClient.UploadFile(req.Context())
	if err != nil {
		return nil, fmt.Errorf("start upload file grpc stream: %w", err)
	}

	err = stream.Send(&files.UploadFileRequest{
		Data: &files.UploadFileRequest_FileInfo{
			FileInfo: &files.UploadFileRequest_Info{
				Name: header.Filename,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("send file info msg to stream: %w", err)
	}

	chunk := make([]byte, p.uploadFileChunkSize)

	var n int
	for {
		n, err = f.Read(chunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read next chunk of file content: %w", err)
		}

		err = stream.Send(&files.UploadFileRequest{
			Data: &files.UploadFileRequest_FileContentChunk{
				FileContentChunk: chunk[:n],
			},
		})
		if err != nil {
			return nil, fmt.Errorf("send chunk of file content to stream: %w", err)
		}
	}

	resp, err = stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("close stream and wait complete on server: %w", err)
	}

	return resp, nil
}

const downloadFilePathPattern = "/v1/files/{name}"

func (p *FilesServiceProxy) DownloadFile(resw http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	_, outboundMarshaler := runtime.MarshalerForRequest(p.mux, req)

	ctx, err := runtime.AnnotateContext(req.Context(), p.mux, req, "/example.files.v1.FilesService/DownloadFile", runtime.WithHTTPPathPattern(downloadFilePathPattern))
	if err != nil {
		runtime.HTTPError(ctx, p.mux, outboundMarshaler, resw, req, err)
		return
	}

	if err := p.downloadFile(ctx, resw, req, pathParams); err != nil {
		runtime.HTTPError(ctx, p.mux, outboundMarshaler, resw, req, err)
		return
	}

	gwruntime.ForwardResponseMessage(ctx, p.mux, outboundMarshaler, resw, req, &emptypb.Empty{}, p.mux.GetForwardResponseOptions()...)
}

func (p *FilesServiceProxy) downloadFile(ctx context.Context, resw http.ResponseWriter, req *http.Request, pathParams map[string]string) error {
	stream, err := p.filesServiceClient.DownloadFile(req.Context(), &files.DownloadFileRequest{
		Name: pathParams["name"],
	})
	if err != nil {
		return fmt.Errorf("start stream of download file: %w", err)
	}
	defer stream.CloseSend()

	fileInfoMessage, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("received msg with file info from stream: %w", err)
	}

	fileHeader := fileInfoMessage.GetFileHeader()
	responseHeaders := resw.Header()
	responseHeaders.Add("content-type", fileHeader.GetContentType())
	responseHeaders.Add("content-disposition", fmt.Sprintf("%s; filename=\"%s\"", formFileName, fileHeader.GetName()))

	for {
		chunkMessage, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("received msg with file content chunk from stream: %w", err)
		}

		_, err = resw.Write(chunkMessage.GetFileContentChunk())
		if err != nil {
			return fmt.Errorf("write chunk of file content to response: %w", err)
		}
	}

	return nil
}
