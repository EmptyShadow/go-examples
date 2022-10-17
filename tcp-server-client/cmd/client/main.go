package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

func main() {
	log.Println("start client")

	targetAddress := flag.String("target-address", ":9999", "address of target tcp server")
	dialTimeout := flag.Duration("dial-timeout", time.Second*30, "durection of dial timeout")
	flag.Parse()

	conn, err := net.DialTimeout("tcp", *targetAddress, *dialTimeout)
	if err != nil {
		err = fmt.Errorf("dial tcp client connection to target tcp server: %w", err)
		log.Fatalln(err)
	}
	defer conn.Close()

	log.Println("connect from", conn.LocalAddr().String(), "to", conn.RemoteAddr().String())

	buf := make([]byte, 10)

	for {
		number := rand.Int63()
		log.Println("generate number", number)

		binary.PutVarint(buf, number)

		_, err = conn.Write(buf)
		if err != nil {
			err = fmt.Errorf("send number to server: %w", err)
			log.Println(err)
			break
		}

		_, err = conn.Read(buf)
		if err != nil {
			err = fmt.Errorf("get state from server: %w", err)
			log.Println(err)
			break
		}

		number, n := binary.Varint(buf)
		if n < 0 {
			err = errors.New("read large number")
		}
		if n == 0 {
			err = errors.New("buf is small")
		}
		if err != nil {
			log.Println(err)
			break
		}

		log.Println("State from server:", number)
		time.Sleep(time.Second)
	}

	log.Println("stop client")
}
