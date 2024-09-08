package main

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)



func TestPathTransform(t *testing.T) {
	key := "hello world!"
	pathKey := CASPathTransform(key)
	assert.Equal(t, "430ce/34d02/0724e/d75a1/96dfc/2ad67/c7777/2d169", pathKey.Pathname)
	assert.Equal(t, "430ce34d020724ed75a196dfc2ad67c77772d169", pathKey.Filename)

}

func TestStore(t *testing.T) {
	// create a new store
	opts := StoreOpts{
		PathTransform: CASPathTransform,
	}
	s := NewStore(opts)

	key := "key"

	// write to the store
	data := []byte("hello world!")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	// read from the store
	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, data, b)

	s.Delete(key)
}

func TestDelete(t *testing.T) {
	// create a new store
	opts := StoreOpts{
		PathTransform: CASPathTransform,
	}
	s := NewStore(opts)
	key := "key2"

	// write to the store
	data := []byte("hello world!")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	// sleep for 2 seconds
	time.Sleep(2 * time.Second)

	// delete from the store
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}

}
