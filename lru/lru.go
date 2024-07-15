package lru

import (
	"container/list"
	"sync"
)

type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
}

type entry struct {
	key string
}

var (
	instance *LRUCache
	once     sync.Once
)

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

func GetInstance(capacity int) *LRUCache {
	once.Do(func() {
		instance = NewLRUCache(capacity)
	})
	return instance
}

func (l *LRUCache) Contains(key string) bool {
	if ele, found := l.cache[key]; found {
		l.list.MoveToFront(ele)
		return true
	}
	return false
}

func (l *LRUCache) Put(key string) {
	if ele, found := l.cache[key]; found {
		l.list.MoveToFront(ele)
		return
	}

	if l.list.Len() == l.capacity {
		back := l.list.Back()
		if back != nil {
			l.list.Remove(back)
			delete(l.cache, back.Value.(*entry).key)
		}
	}

	ele := l.list.PushFront(&entry{key})
	l.cache[key] = ele
}
