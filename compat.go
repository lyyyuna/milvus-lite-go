package milvuslite

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-proto/go-api/v2/milvuspb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ListCollections works around a milvus-lite bug where ShowCollections
// does not return collection_ids, causing the Go SDK's ListCollections
// to return an empty list. This function parses collection_names directly.
func ListCollections(ctx context.Context, addr string) ([]string, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	client := milvuspb.NewMilvusServiceClient(conn)
	resp, err := client.ShowCollections(ctx, &milvuspb.ShowCollectionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("ShowCollections: %w", err)
	}

	return resp.GetCollectionNames(), nil
}
