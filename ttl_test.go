package memcache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetEx(t *testing.T) {
	ctx := context.Background()

	c, err := NewCacheWithSize[[]byte](50)
	require.NoError(t, err)

	{
		keys := []string{}
		for i := 0; i < 20; i++ {
			key := fmt.Sprintf("key-%d", i)

			// SetEx with time 0 is the same as just a Set, because there is no expiry time
			// aka, the key doesn't expire.
			err := c.SetEx(ctx, key, []byte("a"), time.Duration(0))
			require.NoError(t, err)
			keys = append(keys, key)
		}

		for _, key := range keys {
			buf, exists, err := c.Get(ctx, key)
			require.True(t, exists)
			require.NoError(t, err)
			require.NotNil(t, buf)

			exists, err = c.Exists(ctx, key)
			require.NoError(t, err)
			require.True(t, exists)
		}

		values, batchExists, err := c.BatchGet(ctx, keys)
		require.NoError(t, err)
		for i := range values {
			require.NotNil(t, values[i])
			require.True(t, batchExists[i])
		}
	}

	{
		keys := []string{}
		for i := 0; i < 20; i++ {
			key := fmt.Sprintf("key-%d", i)
			err := c.SetEx(ctx, key, []byte("a"), time.Second*10) // a key that expires in 10 seconds
			require.NoError(t, err)
			keys = append(keys, key)
		}

		for _, key := range keys {
			buf, exists, err := c.Get(ctx, key)
			require.NoError(t, err)
			require.NotNil(t, buf)
			require.True(t, exists)

			exists, err = c.Exists(ctx, key)
			require.NoError(t, err)
			require.True(t, exists)
		}

		values, batchExists, err := c.BatchGet(ctx, keys)
		require.NoError(t, err)

		for i := range values {
			require.NotNil(t, values[i])
			require.True(t, batchExists[i])
		}
	}
}
