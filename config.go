package memcache

import cachestore "github.com/goware/cachestore2"

type Config struct {
	cachestore.StoreOptions
	Size int
}

func (c *Config) Apply(options *cachestore.StoreOptions) {
	c.StoreOptions.Apply(options)
}
