package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/EmptyShadow/go-examples/grpc-files/pb/files/v1"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const tcpAddress = "0.0.0.0:8080"

func main() {
	serverAddr := flag.String("server-address", "localhost:9000", "address of grpc server")
	dialTimeout := flag.Duration("dial-timeout", time.Second*30, "timeout of wait dial connect to server")
	flag.Parse()

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

	marshler := &gwruntime.JSONPb{}
	mux := gwruntime.NewServeMux(gwruntime.WithMarshalerOption("*", marshler))

	filesServiceClient := files.NewFilesServiceClient(conn)
	filesServiceProxy := NewFilesServiceProxy(filesServiceClient, mux)

	files.RegisterFilesServiceHandlerClient(context.TODO(), mux, filesServiceClient)
	filesServiceProxy.RegistrationHTTP(mux) // Для того чтобы переопределить методы основного FilesServiceGateway.

	tcpListener, err := net.Listen("tcp", tcpAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer tcpListener.Close()

	server := &http.Server{Handler: mux}

	log.Println("listen", tcpAddress)

	if err = server.Serve(tcpListener); err != nil {
		log.Fatalln(err)
	}
}
