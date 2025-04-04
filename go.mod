module github.com/goware/cachestore-mem

go 1.23.0

replace github.com/goware/cachestore2 => ../cachestore2

require (
	github.com/elastic/go-freelru v0.16.0
	github.com/goware/cachestore2 v0.0.0-00010101000000-000000000000
	github.com/goware/singleflight v0.3.0
	github.com/stretchr/testify v1.10.0
	github.com/zeebo/xxh3 v1.0.2
	golang.org/x/sync v0.12.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
