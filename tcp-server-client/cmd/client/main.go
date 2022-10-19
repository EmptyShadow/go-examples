package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	tcp_server_client "github.com/EmptyShadow/go-examples/tcp-server-client"
)

func main() {
	log.Println("start client")

	targetAddress := flag.String("target-address", ":9999", "address of target tcp server")
	dialTimeout := flag.Duration("dial-timeout", time.Second*30, "durection of dial timeout")
	repeatCount := flag.Int("repeat-count", 5, "repeat count")
	flag.Parse()

	conn, err := net.DialTimeout("tcp", *targetAddress, *dialTimeout)
	if err != nil {
		err = fmt.Errorf("dial tcp client connection to target tcp server: %w", err)
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Println("connect from", conn.LocalAddr().String(), "to", conn.RemoteAddr().String())

	protocol := tcp_server_client.NewProtocol(conn)

	for i := 0; i < *repeatCount; i++ {
		number := rand.Int63()
		log.Println("generate number", number)

		if err = protocol.WriteNumber(number); err != nil {
			err = fmt.Errorf("write number: %w", err)
			log.Println(err)
			break
		}

		number, err = protocol.ReadNumber()
		if err != nil {
			err = fmt.Errorf("read number: %w", err)
			log.Println(err)
			break
		}

		log.Println("sum of squares:", number)
		time.Sleep(time.Second)
	}

	log.Println("stop client")
}
