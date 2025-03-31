module github.com/goware/cachestore-mem

go 1.24.1

replace github.com/goware/cachestore => ../cachestore

require (
	github.com/elastic/go-freelru v0.16.0
	github.com/goware/cachestore v0.11.0
	github.com/goware/singleflight v0.3.0
	github.com/stretchr/testify v1.10.0
	github.com/zeebo/xxh3 v1.0.2
	golang.org/x/sync v0.12.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
