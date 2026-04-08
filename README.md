# milvus-lite-go

Go wrapper for [milvus-lite](https://github.com/milvus-io/milvus-lite) — an embedded vector database.

Start a milvus-lite server from Go with zero external dependencies. The pre-built binary is automatically downloaded from PyPI on first use and cached locally.

## Install

```bash
go get github.com/lyyyuna/milvus-lite-go
```

## Usage

```go
package main

import (
	"context"
	"log"

	milvuslite "github.com/lyyyuna/milvus-lite-go"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func main() {
	// Start milvus-lite (downloads binary on first run)
	server, err := milvuslite.Start("./milvus.db")
	if err != nil {
		log.Fatal(err)
	}
	defer server.Stop()

	// Connect with the official Go SDK
	ctx := context.Background()
	c, _ := client.NewClient(ctx, client.Config{Address: server.Addr()})
	defer c.Close()

	// Use as normal Milvus
	schema := entity.NewSchema().WithName("demo").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(128))

	c.CreateCollection(ctx, schema, entity.DefaultShardNumber)
}
```

## How it works

1. On first `Start()`, downloads the milvus-lite wheel from PyPI (respects `pip.conf` mirror settings for users in China)
2. Extracts the `milvus` binary + shared libraries to `~/.cache/milvus-lite/{version}/{os}-{arch}/`
3. Starts the binary as a subprocess listening on a random localhost port
4. Returns the gRPC address for use with [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go)

## Supported platforms

| OS | Arch | Status |
|----|------|--------|
| macOS | arm64 (Apple Silicon) | ✅ |
| macOS | amd64 (Intel) | ✅ |
| Linux | amd64 | ✅ |
| Linux | arm64 | ✅ |

## API compatibility

This wrapper uses the same gRPC protocol as Milvus, so the official [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go) works out of the box. All milvus-lite supported APIs work:

- **Collection**: Create, Drop, Has, Describe, GetCollectionStatistics
- **Index**: Create (FLAT, IVF_FLAT), Describe, Drop
- **Data**: Insert, Upsert, Delete
- **Search**: Search (with filters), Query
- **Load**: Load, Release, GetLoadState

### Known issues

**`client.ListCollections()` returns empty** — This is a [milvus-lite bug](https://github.com/milvus-io/milvus-lite): `ShowCollections` doesn't return `collection_ids`, which the Go SDK relies on. Use the provided workaround:

```go
// Instead of: colls, _ := client.ListCollections(ctx)
// Use:
names, _ := milvuslite.ListCollections(ctx, server.Addr())
```

**Unsupported features** (same as milvus-lite limitations):
- Partitions
- RBAC (users/roles)
- Aliases

## License

Apache 2.0
