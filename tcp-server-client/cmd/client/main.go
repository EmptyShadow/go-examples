package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	log.Println("start client")

	targetAddress := flag.String("target-address", ":9999", "address of target tcp server")
	flag.Parse()

	netAddress, err := net.ResolveTCPAddr("tcp", *targetAddress)
	if err != nil {
		err = fmt.Errorf("resolve tcp address by %s: %w", *targetAddress, err)
		log.Fatalln(err)
	}

	conn, err := net.DialTCP("tcp", nil, netAddress)
	if err != nil {
		err = fmt.Errorf("dial tcp client connection to target tcp server: %w", err)
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Println("connect from", conn.LocalAddr().String(), "to", conn.RemoteAddr().String())

	buf := make([]byte, 128)

	for {
		n, err := conn.Read(buf)
		if errors.Is(err, io.EOF) {
			log.Println("connection is closed")
			break
		}
		if err != nil {
			log.Fatalln("read data from connection: %w", err)
		}

		log.Println("read data", string(buf[:n]))
	}

	log.Println("stop client")
}
