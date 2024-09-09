package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
)

const defaultRootFolderName = "storage"

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) fileNameWithPath() string {
	return path.Join(p.Pathname, p.Filename)
}

func DefaultPathTransform(key string) PathKey {
	return PathKey{
		Pathname: defaultRootFolderName,
		Filename: key,
	}
}

func CASPathTransform(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	if len(hashStr)%blockSize != 0 {
		sliceLen++
	}

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		start := i * blockSize
		end := (i + 1) * blockSize
		if end > len(hashStr) {
			end = len(hashStr)
		}
		paths[i] = hashStr[start:end]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

type StoreOpts struct {
	Root          string
	PathTransform PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransform == nil {
		opts.PathTransform = DefaultPathTransform
	}

	if opts.Root == "" {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// check if the file exists
func (s *Store) Exists(key string) bool {
	pathKey := s.PathTransform(key)
	fileNameWithPath := path.Join(s.Root, pathKey.fileNameWithPath())
	_, err := os.Stat(fileNameWithPath)
	return err != fs.ErrNotExist
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransform(key)
	filepath := path.Join(s.Root, pathKey.Pathname)

	return os.RemoveAll(strings.Join(strings.Split(filepath, "/")[:2], "/"))
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.ReadStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, f)

	if err != nil {
		return nil, err
	}
	log.Printf("read %d bytes from %s", n, key)
	return buf, nil
}

func (s *Store) ReadStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransform(key)
	fileNameWithPath := path.Join(s.Root, pathKey.fileNameWithPath())
	f, err := os.Open(fileNameWithPath)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransform(key)
	filepath := path.Join(s.Root, pathKey.Pathname)

	if err := os.MkdirAll(filepath, os.ModePerm); err != nil {
		return err
	}

	fileNameWithPath := path.Join(s.Root, pathKey.fileNameWithPath())
	f, err := os.Create(fileNameWithPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("wrote %d bytes to %s", n, fileNameWithPath)

	return nil
}
