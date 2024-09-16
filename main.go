package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
	FileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr[1:] + "_network",
		PathTransformFunc: CASPathTransform,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(FileServerOpts)

	// tcp transport will push peers to the server
	tcpTransport.PushPeer = s.AddPeer

	return s
}

func main() {

	s1 := makeServer(":4001", "")
	s2 := makeServer(":4002", "")

	s3 := makeServer(":4003", ":4001", ":4002")

	go func() {
		if err := s1.Start(); err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		if err := s2.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(2 * time.Second)

	go s3.Start()

	time.Sleep(2 * time.Second)

	// data := bytes.NewReader([]byte("Lorem Ipsum is simply dummy text of the printing and typesetting industry."))
	// s3.StoreData("hello.txt", data)
	_, r, err := s3.ReadData("hello.txt")

	if err != nil {
		fmt.Println("Oh no!")
		fmt.Println(err.Error())
	}

	buff := new(bytes.Buffer)
	io.Copy(buff, r)
	fmt.Print("recieved from server::: ", buff.String())
	select {}

	// go func() {
	// 	time.Sleep(5 * time.Second)
	// 	server.Stop()
	// }()

}
