package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
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

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransform(key)
	filepath := path.Join(s.Root, pathKey.Pathname)

	return os.RemoveAll(strings.Join(strings.Split(filepath, "/")[:2], "/"))
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.ReadStream(key)
}

func (s *Store) ReadStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransform(key)
	fileNameWithPath := path.Join(s.Root, pathKey.fileNameWithPath())
	file, err := os.Open(fileNameWithPath)
	if err != nil {
		return 0, nil, err
	}
	fi, err := file.Stat()

	return fi.Size(), file, nil
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
