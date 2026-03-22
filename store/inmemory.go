package store

import "sync"

type InMemoryStore[T any] struct {
	mu    sync.RWMutex
	items map[string]T
}

func NewInMemoryStore[T any]() *InMemoryStore[T] {
	return &InMemoryStore[T]{items: make(map[string]T)}
}

func compositeKey(partitionKey, sortKey string) string {
	return partitionKey + "\x00" + sortKey
}

func (s *InMemoryStore[T]) Get(partitionKey, sortKey string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[compositeKey(partitionKey, sortKey)]
	return v, ok
}

func (s *InMemoryStore[T]) Set(partitionKey, sortKey string, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[compositeKey(partitionKey, sortKey)] = value
}

func (s *InMemoryStore[T]) Delete(partitionKey, sortKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, compositeKey(partitionKey, sortKey))
}

func (s *InMemoryStore[T]) Pop(partitionKey, sortKey string) (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := compositeKey(partitionKey, sortKey)
	v, ok := s.items[k]
	if ok {
		delete(s.items, k)
	}
	return v, ok
}
