package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity < 0 {
		capacity = 0
	}
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]V),
	}
}
func (c *Cache[K, V]) Get(k K) (v V, exist bool) {
	v, exist = c.items[k]
	return v, exist
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	_, exist := c.items[k]; 
	if exist {
		c.items[k] = v
		return true
	}
	if len(c.items) >= c.capacity {
		return false
	}
	c.items[k] = v
	return true
}
