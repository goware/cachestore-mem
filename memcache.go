package memcache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	cachestore "github.com/goware/cachestore2"
	"github.com/goware/singleflight"
	"github.com/maypok86/otter/v2"
)

func NewBackend(size uint32, opts ...cachestore.StoreOptions) (cachestore.Backend, error) {
	return NewCacheWithSize[any](size, opts...)
}

func NewCache[V any](opts ...cachestore.StoreOptions) (cachestore.Store[V], error) {
	const defaultLRUSize = 512
	return NewCacheWithSize[V](defaultLRUSize, opts...)
}

func NewCacheWithSize[V any](size uint32, opts ...cachestore.StoreOptions) (*MemLRU[V], error) {
	if size == 0 {
		return nil, errors.New("cachestore-mem: size cannot be 0")
	}

	cache, err := otter.New[string, V](&otter.Options[string, V]{
		MaximumSize:      int(size),
		ExpiryCalculator: otter.ExpiryWriting[string, V](0), // enable expiry, actual TTL set per-key
	})
	if err != nil {
		return nil, err
	}

	options := cachestore.ApplyOptions(opts...)
	if options.LockRetryTimeout == 0 {
		options.LockRetryTimeout = 8 * time.Second
	}

	memLRU := &MemLRU[V]{
		options: options,
		cache:   cache,
	}

	return memLRU, nil
}

type MemLRU[V any] struct {
	options      cachestore.StoreOptions
	cache        *otter.Cache[string, V]
	singleflight singleflight.Group[string, V]
}

var _ cachestore.Store[any] = &MemLRU[any]{}

func (m *MemLRU[V]) Name() string {
	return "memcache"
}

func (m *MemLRU[V]) Options() cachestore.StoreOptions {
	return m.options
}

func (m *MemLRU[V]) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.cache.GetIfPresent(key)
	return ok, nil
}

func (m *MemLRU[V]) Set(ctx context.Context, key string, value V) error {
	return m.SetEx(ctx, key, value, m.options.DefaultKeyExpiry)
}

func (m *MemLRU[V]) SetEx(ctx context.Context, key string, value V, ttl time.Duration) error {
	if err := m.setKeyValue(key, value, ttl); err != nil {
		return err
	}
	return nil
}

func (m *MemLRU[V]) BatchSet(ctx context.Context, keys []string, values []V) error {
	return m.BatchSetEx(ctx, keys, values, m.options.DefaultKeyExpiry)
}

func (m *MemLRU[V]) BatchSetEx(ctx context.Context, keys []string, values []V, ttl time.Duration) error {
	if len(keys) != len(values) {
		return errors.New("cachestore-mem: keys and values are not the same length")
	}
	if len(keys) == 0 {
		return errors.New("cachestore-mem: no keys are passed")
	}
	for i, key := range keys {
		m.setKeyValue(key, values[i], ttl)
	}
	return nil
}

func (m *MemLRU[V]) Get(ctx context.Context, key string) (V, bool, error) {
	var out V
	v, ok := m.cache.GetIfPresent(key)
	if !ok {
		return out, false, nil
	}
	return v, true, nil
}

func (m *MemLRU[V]) BatchGet(ctx context.Context, keys []string) ([]V, []bool, error) {
	vals := make([]V, 0, len(keys))
	oks := make([]bool, 0, len(keys))
	var out V

	for _, key := range keys {
		v, ok := m.cache.GetIfPresent(key)
		if !ok {
			vals = append(vals, out)
			oks = append(oks, false)
			continue
		}
		vals = append(vals, v)
		oks = append(oks, true)
	}

	return vals, oks, nil
}

func (m *MemLRU[V]) Delete(ctx context.Context, key string) error {
	m.cache.Invalidate(key)
	return nil
}

func (m *MemLRU[V]) DeletePrefix(ctx context.Context, keyPrefix string) error {
	var toDelete []string
	for key := range m.cache.All() {
		if strings.HasPrefix(key, keyPrefix) {
			toDelete = append(toDelete, key)
		}
	}
	for _, key := range toDelete {
		m.cache.Invalidate(key)
	}
	return nil
}

func (m *MemLRU[V]) ClearAll(ctx context.Context) error {
	m.cache.InvalidateAll()
	return nil
}

func (m *MemLRU[V]) GetOrSetWithLock(ctx context.Context, key string, getter func(context.Context, string) (V, error)) (V, error) {
	return m.GetOrSetWithLockEx(ctx, key, getter, m.options.DefaultKeyExpiry)
}

func (m *MemLRU[V]) GetOrSetWithLockEx(
	ctx context.Context, key string, getter func(context.Context, string) (V, error), ttl time.Duration,
) (V, error) {
	var out V

	ctx, cancel := context.WithTimeout(ctx, m.options.LockRetryTimeout)
	defer cancel()

	v, ok := m.cache.GetIfPresent(key)
	if ok {
		return v, nil
	}

	v, err, _ := m.singleflight.Do(key, func() (V, error) {
		v, err := getter(ctx, key)
		if err != nil {
			return out, fmt.Errorf("cachestore-mem: getter error: %w", err)
		}
		if err := m.setKeyValue(key, v, ttl); err != nil {
			return out, err
		}
		return v, nil
	})
	if err != nil {
		return out, fmt.Errorf("cachestore-mem: singleflight error: %w", err)
	}

	return v, nil
}

func (m *MemLRU[V]) setKeyValue(key string, value V, ttl time.Duration) error {
	if len(key) > cachestore.MaxKeyLength {
		return cachestore.ErrKeyLengthTooLong
	}
	if len(key) == 0 {
		return cachestore.ErrInvalidKey
	}
	m.cache.Set(key, value)
	if ttl > 0 {
		m.cache.SetExpiresAfter(key, ttl)
	}
	return nil
}
