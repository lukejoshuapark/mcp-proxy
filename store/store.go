package store

type Store[T any] interface {
	Get(partitionKey, sortKey string) (T, bool)
	Set(partitionKey, sortKey string, value T)
	Delete(partitionKey, sortKey string)
	Pop(partitionKey, sortKey string) (T, bool)
}
