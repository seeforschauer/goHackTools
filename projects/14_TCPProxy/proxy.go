package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

var (
	localAddr  = flag.String("l", "localhost:9999", "local address")
	remoteAddr = flag.String("r", "localhost:80", "remote address")
)

func main() {
	flag.Parse()

	fmt.Printf("Listening: %v\nProxying: %v\n\n", *localAddr, *remoteAddr)

	addr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		log.Fatalln(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}

	pending, complete := make(chan *net.TCPConn), make(chan *net.TCPConn)

	for i := 0; i < 5; i++ {
		go handleConn(pending, complete)
	}

	go closeConn(complete)

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatalln(err)
		}
		pending <- conn
	}
}

func proxyConn(conn *net.TCPConn) {
	rAddr, err := net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		log.Fatalln(err)
	}

	rConn, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer rConn.Close()

	buf := &bytes.Buffer{}

	for {
		data := make([]byte, 256)

		n, err := conn.Read(data)
		if err != nil {
			log.Println(err)
			return
		}

		buf.Write(data[:n])

		if data[0] == '\r' && data[1] == '\n' {
			break
		}
	}

	if _, err := rConn.Write(buf.Bytes()); err != nil {
		log.Println(err)
		return
	}

	log.Printf("sent:\n%v", hex.Dump(buf.Bytes()))

	data := make([]byte, 1024)

	n, err := rConn.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
			return
		}

		log.Printf("received err: %v", err)
	}

	log.Printf("received:\n%v", hex.Dump(data[:n]))
}

func handleConn(in <-chan *net.TCPConn, out chan<- *net.TCPConn) {
	for conn := range in {
		proxyConn(conn)
		out <- conn
	}
}

func closeConn(in <-chan *net.TCPConn) {
	for conn := range in {
		conn.Close()
	}
}
