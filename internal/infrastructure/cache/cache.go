package cache

import "context"

// здесь мог бы быть redis или какой-то другой кеш
type KeyValueCache[K comparable, V any] interface {
	Get(ctx context.Context, key K) (v V, ok bool)
	Set(ctx context.Context, key K, v V)
	Delete(ctx context.Context, key K)
}
