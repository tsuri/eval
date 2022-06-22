package hash

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"reflect"
)

type Hasher interface {
	Hash(o any) []byte
}

type BuildConfig struct {
	N int64
}

type digest []byte

type HashState struct {
	RunningHash hash.Hash
}

func NewHashState() *HashState {
	return &HashState{
		RunningHash: sha256.New(),
	}
}

func (h HashState) Digest() digest {
	return h.RunningHash.Sum(nil)
}

func Hash(o any) digest {
	s := NewHashState()

	fmt.Printf("type: %T\nvalue: %v\n", o, o)

	vo := reflect.ValueOf(o)
	fmt.Printf("type: %T\nvalue: %v\n", vo, vo)

	return s.Digest()
}
