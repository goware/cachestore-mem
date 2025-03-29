module github.com/goware/cachestore-mem

go 1.24.1

replace github.com/goware/cachestore => ../cachestore

require (
	github.com/goware/cachestore v0.11.0
	github.com/goware/singleflight v0.3.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/stretchr/testify v1.10.0
	golang.org/x/sync v0.12.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
