package main

import (
	"fmt"
	"io"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts
	store  *Store
	quitCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:          opts.StorageRoot,
		PathTransform: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

func (s *FileServer) Stop() {
	close(s.quitCh)
	fmt.Println("server signalled to stop")
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	s.loop()

	return nil
}

func (s *FileServer) loop() {
	defer s.Transport.Close()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():
			fmt.Println("rpc received: ", rpc)
			// handle rpc
		}
	}
}

func (s *FileServer) Store(key string, r io.Reader) error {

	return nil
}
