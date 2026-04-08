package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	milvuslite "github.com/lyyyuna/milvus-lite-go/v2"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func main() {
	// Start milvus-lite server
	server, err := milvuslite.Start("./milvus_demo.db")
	if err != nil {
		log.Fatalf("start milvus-lite: %v", err)
	}
	defer server.Stop()

	fmt.Printf("milvus-lite running at %s\n", server.Addr())

	// Connect using official milvus-sdk-go
	ctx := context.Background()
	c, err := client.NewClient(ctx, client.Config{Address: server.Addr()})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer c.Close()

	// Create collection
	dim := 128
	schema := entity.NewSchema().WithName("demo").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(dim)))

	if err := c.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		log.Fatalf("create collection: %v", err)
	}

	// Insert vectors
	vectors := make([][]float32, 100)
	for i := range vectors {
		v := make([]float32, dim)
		for j := range v {
			v[j] = rand.Float32()
		}
		vectors[i] = v
	}

	if _, err := c.Insert(ctx, "demo", "", entity.NewColumnFloatVector("vector", dim, vectors)); err != nil {
		log.Fatalf("insert: %v", err)
	}

	// Create index and load
	idx, _ := entity.NewIndexFlat(entity.L2)
	if err := c.CreateIndex(ctx, "demo", "vector", idx, false); err != nil {
		log.Fatalf("create index: %v", err)
	}
	if err := c.LoadCollection(ctx, "demo", false); err != nil {
		log.Fatalf("load: %v", err)
	}

	// Search
	searchVec := []entity.Vector{entity.FloatVector(vectors[0])}
	result, err := c.Search(ctx, "demo", nil, "", []string{}, searchVec, "vector", entity.L2, 5, nil)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	fmt.Printf("search returned %d results\n", result[0].ResultCount)
	for i := 0; i < result[0].ResultCount; i++ {
		fmt.Printf("  id=%v score=%v\n", result[0].IDs.(*entity.ColumnInt64).Data()[i], result[0].Scores[i])
	}
}
