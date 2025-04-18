package memcache

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	cachestore "github.com/goware/cachestore2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestCacheInt(t *testing.T) {
	c, err := NewCacheWithSize[int](50, cachestore.WithDefaultKeyExpiry(12*time.Second))
	require.NoError(t, err)
	require.True(t, c.options.DefaultKeyExpiry.Seconds() == 12)

	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		c.Set(ctx, fmt.Sprintf("i%d", i), i)
	}
	for i := 0; i < 10; i++ {
		v, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %d to be in cache", i)
		}
		if v != i {
			t.Errorf("expected %d to be %d", v, i)
		}
	}
	for i := 0; i < 10; i++ {
		c.Delete(ctx, fmt.Sprintf("i%d", i))
	}
	for i := 0; i < 10; i++ {
		_, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %d to not be in cache", i)
		}
	}
}

func TestCacheString(t *testing.T) {
	c, err := NewCacheWithSize[string](50)
	require.NoError(t, err)
	require.True(t, c.options.DefaultKeyExpiry == 0)

	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		c.Set(ctx, fmt.Sprintf("i%d", i), fmt.Sprintf("v%d", i))
	}
	for i := 0; i < 10; i++ {
		exists, err := c.Exists(ctx, fmt.Sprintf("i%d", i))
		require.NoError(t, err)
		require.True(t, exists)

		v, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %d to be in cache", i)
		}
		if v != fmt.Sprintf("v%d", i) {
			t.Errorf("expected %s to be %s", v, fmt.Sprintf("v%d", i))
		}
	}
	for i := 0; i < 10; i++ {
		c.Delete(ctx, fmt.Sprintf("i%d", i))
	}
	for i := 0; i < 10; i++ {
		_, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %d to not be in cache", i)
		}
	}
}

func TestCacheObject(t *testing.T) {
	type custom struct {
		value int
		data  string
	}

	c, err := NewCacheWithSize[custom](50)
	ctx := context.Background()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		c.Set(ctx, fmt.Sprintf("i%d", i), custom{i, fmt.Sprintf("v%d", i)})
	}
	for i := 0; i < 10; i++ {
		v, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %d to be in cache", i)
		}
		value := custom{
			i,
			fmt.Sprintf("v%d", i),
		}
		if v != value {
			t.Errorf("expected %v to be %v", v, custom{i, fmt.Sprintf("v%d", i)})
		}
	}
	for i := 0; i < 10; i++ {
		c.Delete(ctx, fmt.Sprintf("i%d", i))
	}
	for i := 0; i < 10; i++ {
		l, _, err := c.Get(ctx, fmt.Sprintf("i%d", i))
		if err != nil {
			t.Errorf("expected %v to not be in cache", l)
		}
	}
}

type obj struct {
	A string
	B string
}

func TestBasicBatchObjects(t *testing.T) {
	cache, err := NewCacheWithSize[*obj](50)
	if err != nil {
		t.Fatal(err)
	}

	var keys = []string{
		"test-obj3-a", "test-obj3-b",
	}

	var in = []*obj{
		{A: "3a", B: "3a"},
		{A: "3b", B: "3b"},
	}

	ctx := context.Background()
	err = cache.BatchSet(ctx, keys, in)
	require.NoError(t, err)

	// adding some keys which will not exist
	fetchKeys := []string{"no1"}
	fetchKeys = append(fetchKeys, keys...)
	fetchKeys = append(fetchKeys, []string{"no2", "no3"}...)

	out, exists, err := cache.BatchGet(ctx, fetchKeys)
	require.NoError(t, err)
	require.Equal(t, []*obj{nil, in[0], in[1], nil, nil}, out)
	require.Equal(t, []bool{false, true, true, false, false}, exists)
}

func TestExpiryOptions(t *testing.T) {
	ctx := context.Background()

	cache, err := NewCache[string](cachestore.WithDefaultKeyExpiry(1 * time.Second))
	require.NoError(t, err)

	rcache, ok := cache.(*MemLRU[string])
	require.True(t, ok)
	require.True(t, rcache.options.DefaultKeyExpiry.Seconds() == 1)

	err = cache.Set(ctx, "hi", "bye")
	require.NoError(t, err)

	err = cache.SetEx(ctx, "another", "longer", 20*time.Second)
	require.NoError(t, err)

	value, exists, err := cache.Get(ctx, "hi")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "bye", value)

	// pause to wait for expiry.. we have to wait at least 5 seconds
	// as memLRU does expiry cycles that amount of time
	time.Sleep(6 * time.Second)

	value, exists, err = cache.Get(ctx, "hi")
	require.NoError(t, err)
	require.False(t, exists)
	require.Equal(t, "", value)

	value, exists, err = cache.Get(ctx, "another")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "longer", value)
}

func TestGetOrSetWithLock(t *testing.T) {
	ctx := context.Background()

	cache, err := NewCache[string]()
	require.NoError(t, err)

	var counter atomic.Uint32
	getter := func(ctx context.Context, key string) (string, error) {
		counter.Add(1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return "result:" + key, nil
		}
	}

	concurrentCalls := 15
	results := make(chan string, concurrentCalls)
	key := fmt.Sprintf("concurrent-%d", rand.Uint64())

	var wg errgroup.Group
	for i := 0; i < concurrentCalls; i++ {
		wg.Go(func() error {
			v, err := cache.GetOrSetWithLock(ctx, key, getter)
			if err != nil {
				return err
			}
			results <- v
			return nil
		})
	}

	require.NoError(t, wg.Wait())
	assert.Equalf(t, 1, int(counter.Load()), "getter should be called only once")

	for i := 0; i < concurrentCalls; i++ {
		select {
		case v := <-results:
			assert.Equal(t, "result:"+key, v)
		default:
			t.Errorf("expected %d results but only got %d", concurrentCalls, i)
		}
	}
}

func TestBackend(t *testing.T) {
	backend, err := NewBackend(500)
	require.NoError(t, err)

	cache := cachestore.OpenStore[string](backend)

	{
		err = cache.Set(context.Background(), "key", "value")
		require.NoError(t, err)

		value, exists, err := cache.Get(context.Background(), "key")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "value", value)

		err = cache.Delete(context.Background(), "key")
		require.NoError(t, err)

		value, exists, err = cache.Get(context.Background(), "key")
		require.NoError(t, err)
		require.False(t, exists)
		require.Equal(t, "", value)
	}

	{
		keys := []string{"key1", "key2", "key3"}
		values := []string{"value1", "value2", "value3"}
		err = cache.BatchSet(context.Background(), keys, values)
		require.NoError(t, err)

		batch, exists, err := cache.BatchGet(context.Background(), keys)
		require.NoError(t, err)
		require.Equal(t, values, batch)
		require.Equal(t, []bool{true, true, true}, exists)

		err = cache.DeletePrefix(context.Background(), "key")
		require.NoError(t, err)

		batch, exists, err = cache.BatchGet(context.Background(), keys)
		require.NoError(t, err)
		require.Equal(t, []string{"", "", ""}, batch)
		require.Equal(t, []bool{false, false, false}, exists)
	}
}

func TestBackendGetOrSetWithLock(t *testing.T) {
	backend, err := NewBackend(500)
	require.NoError(t, err)

	cache := cachestore.OpenStore[string](backend)

	ctx := context.Background()

	var counter atomic.Uint32
	getter := func(ctx context.Context, key string) (string, error) {
		counter.Add(1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return "result:" + key, nil
		}
	}

	concurrentCalls := 15
	results := make(chan string, concurrentCalls)
	key := fmt.Sprintf("concurrent-%d", rand.Uint64())

	var wg errgroup.Group
	for i := 0; i < concurrentCalls; i++ {
		wg.Go(func() error {
			v, err := cache.GetOrSetWithLock(ctx, key, getter)
			if err != nil {
				return err
			}
			results <- v
			return nil
		})
	}

	require.NoError(t, wg.Wait())
	assert.Equalf(t, 1, int(counter.Load()), "getter should be called only once")

	for i := 0; i < concurrentCalls; i++ {
		select {
		case v := <-results:
			assert.Equal(t, "result:"+key, v)
		default:
			t.Errorf("expected %d results but only got %d", concurrentCalls, i)
		}
	}
}
