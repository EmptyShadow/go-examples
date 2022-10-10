package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/EmptyShadow/go-examples/grpc-upload-file/pb/pb/files/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	file := flag.String("filepath", "", "path of uploaded file")
	serverAddr := flag.String("server-address", "localhost:9000", "address of grpc server")
	dialTimeout := flag.Duration("dial-timeout", time.Second*30, "timeout of wait dial connect to server")
	flag.Parse()

	f, err := os.Open(*file)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed open file: %w", err))
	}
	defer f.Close()

	fileStat, err := f.Stat()
	if err != nil {
		log.Fatalln(fmt.Errorf("failed get file stat: %w", err))
	}

	log.Println("Name", fileStat.Name(), "Size", fileStat.Size())

	ctx := context.Background()

	dialCtx, cancelDial := context.WithTimeout(ctx, *dialTimeout)
	defer cancelDial()

	conn, err := grpc.DialContext(
		dialCtx,
		*serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed connect to grpc server: %w", err))
	}
	defer conn.Close()

	filesServiceClient := files.NewFilesServiceClient(conn)

	stream, err := filesServiceClient.UploadFile(ctx)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed start upload file grpc stream: %w", err))
	}

	_, name := filepath.Split(*file)

	err = stream.Send(&files.UploadFileRequest{
		Data: &files.UploadFileRequest_FileInfo{
			FileInfo: &files.UploadFileInfo{
				Name: name,
			},
		},
	})
	if err != nil {
		log.Fatalln(fmt.Errorf("failed send uploaded file info: %w", err))
	}

	chunk := make([]byte, 1024)

	for {
		n, err := f.Read(chunk)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalln(fmt.Errorf("failed read next file content chunk: %w", err))
		}

		err = stream.Send(&files.UploadFileRequest{
			Data: &files.UploadFileRequest_FileContentChunk{
				FileContentChunk: chunk[:n],
			},
		})
		if err != nil {
			log.Fatalln(fmt.Errorf("failed send file content chunk: %w", err))
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalln(fmt.Errorf("failed close grpc stream and receive upload file response: %w", err))
	}

	log.Println(res)
}
