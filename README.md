# milvus-lite-go

Go wrapper for [milvus-lite](https://github.com/milvus-io/milvus-lite) — an embedded vector database.

The milvus-lite binary is embedded in platform-specific Go sub-modules via `go:embed`. `go get` only downloads the binary for your platform — no runtime downloads needed.

## Install

```bash
go get github.com/lyyyuna/milvus-lite-go@v2.5.100
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
	server, err := milvuslite.Start("./milvus.db")
	if err != nil {
		log.Fatal(err)
	}
	defer server.Stop()

	ctx := context.Background()
	c, _ := client.NewClient(ctx, client.Config{Address: server.Addr()})
	defer c.Close()

	schema := entity.NewSchema().WithName("demo").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(128))

	c.CreateCollection(ctx, schema, entity.DefaultShardNumber)
}
```

## How it works

```
github.com/lyyyuna/milvus-lite-go/
├── go.mod                          ← main module (pure Go logic)
├── server.go                       ← Start/Stop/Addr
├── embed.go                        ← extract embedded binary to temp dir
├── platform_darwin_arm64.go        ← //go:build darwin && arm64
├── platform/
│   ├── darwin-arm64/go.mod         ← sub-module with embedded binary
│   ├── darwin-amd64/go.mod
│   ├── linux-amd64/go.mod
│   └── linux-arm64/go.mod
```

Build tags select the right sub-module at compile time. `go get` only downloads the matching platform's binary (~24-55MB), not all platforms.

At runtime, the embedded binary is extracted to a temp directory and started as a subprocess.

## Supported platforms

| OS | Arch | Sub-module |
|----|------|------------|
| macOS | arm64 (Apple Silicon) | `platform/darwin-arm64` |
| macOS | amd64 (Intel) | `platform/darwin-amd64` |
| Linux | amd64 | `platform/linux-amd64` |
| Linux | arm64 | `platform/linux-arm64` |

## API compatibility

All milvus-lite supported APIs work with the official [milvus-sdk-go](https://github.com/milvus-io/milvus-sdk-go):

- **Collection**: Create, Drop, Has, Describe, GetCollectionStatistics
- **Index**: Create (FLAT, IVF_FLAT), Describe, Drop
- **Data**: Insert, Upsert, Delete
- **Search**: Search (with filters), Query
- **Load**: Load, Release, GetLoadState

### Known issues

**`client.ListCollections()` returns empty** — milvus-lite bug: `ShowCollections` doesn't return `collection_ids`. Use the provided workaround:

```go
names, _ := milvuslite.ListCollections(ctx, server.Addr())
```

**Unsupported features** (same as milvus-lite limitations):
- Partitions
- RBAC (users/roles)
- Aliases

## License

Apache 2.0
