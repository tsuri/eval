package cache_test

import (
	"eval/pkg/cache"
	"testing"
)

type S struct {
	D int
}

func BenchmarkBase(b *testing.B) {
	c := cache.NewCache[int, string]()

	for i := 0; i < b.N; i++ {
		c.Set(10, "foo")
	}
}

func BenchmarkMultiBase(b *testing.B) {
	c := cache.NewMultiCache[int, string]()

	for i := 0; i < b.N; i++ {
		c.Set(10, "foo")
	}
}

func TestMultiCache(t *testing.T) {
	c := cache.NewMultiCache[int, string]()

	c.Set(10, "foo")
	c.Set(10, "bar")
	c.Set(10, "baz")

	if c.Get(10) != "bar" {
		t.Fatalf("Expecting 'foo'")
	}
}
