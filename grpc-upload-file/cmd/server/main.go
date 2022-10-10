package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

const tcpAddress = "0.0.0.0:9000"

func main() {
	tcpListener, err := net.Listen("tcp", tcpAddress)
	if err != nil {
		log.Fatalln(err)
	}

	server := grpc.NewServer()

	localFileSystem := MustNewLocalFileSystem(os.Getenv("LOCAL_FILE_SYSTEM_ROOT"))
	filesService := NewFilesService(localFileSystem)
	filesServiceServer := NewFilesServiceServer(filesService)
	filesServiceServer.RegistrationGRPC(server)

	log.Println("listen", tcpAddress)

	if err = server.Serve(tcpListener); err != nil {
		log.Fatalln(err)
	}
}
