package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return value, false
	}
	el, found := c.items[key]
	if !found {
		return value, false
	}
	c.ll.MoveToFront(el)
	e := el.Value.(*entry[K, V])
	return e.value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	if el, found := c.items[key]; found {
		el.Value.(*entry[K, V]).value = value
		return
	}
	if c.ll.Len() >= c.capacity {
		old := c.ll.Back()
		ent := old.Value.(*entry[K, V])
		delete(c.items, ent.key)
		c.ll.Remove(old)
	}
	el := c.ll.PushFront(&entry[K, V]{key: key, value: value})
	c.items[key] = el
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
