package cache

import (
	"hash"
	"sort"

	"golang.org/x/exp/constraints"
)

// type Cache[K comparable, V any] interface {
// 	Get(K) (V, bool)
// 	Put(K, V)
// }

// Cache config
//
// KeyMap : a function mapping a key to a cache slot
// ValueFilter: a predicate to filter out acceptable values
// ValueOrder: a sorting function for ordering acceptable values
//
// somewhere we need to specify how many values we return.
// for us is typically one, but 'all' is also useful.

type Document interface {
	Digest() hash.Hash
	Compatible(other Document) bool
}

type Cache[K comparable, V any] struct {
	Data map[K]V
}

func (c *Cache[K, V]) AddValue(key K, value V) {
	c.Data[key] = value
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	c := Cache[K, V]{
		Data: make(map[K]V),
	}
	return &c
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.AddValue(key, value)
}

type MultiCache[K comparable, V any] struct {
	//Cache[K, []V]
	Data map[K][]V
	Sort func([]V)
}

func sortSlice[T constraints.Ordered](s []T) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}

func (c *MultiCache[K, V]) AddValue(key K, value V) {
	c.Data[key] = append(c.Data[key], value)
	c.Sort(c.Data[key])
}

func (c *MultiCache[K, V]) Set(key K, value V) {
	c.AddValue(key, value)
}

func (c *MultiCache[K, V]) Get(key K) V {
	return c.Data[key][0]
}

func NewMultiCache[K comparable, V constraints.Ordered]() *MultiCache[K, V] {
	return &MultiCache[K, V]{
		Data: make(map[K][]V),
		Sort: sortSlice[V],
	}
}
