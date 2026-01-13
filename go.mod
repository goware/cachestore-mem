module github.com/goware/cachestore-mem

go 1.24.0

// replace github.com/goware/cachestore2 => ../cachestore2

require (
	github.com/goware/cachestore2 v0.12.2
	github.com/goware/singleflight v0.3.0
	github.com/maypok86/otter/v2 v2.3.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/sync v0.13.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
