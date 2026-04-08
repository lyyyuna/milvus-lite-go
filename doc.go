// Package milvuslite provides a Go wrapper for milvus-lite, an embedded vector database.
//
// It embeds the pre-built milvus-lite binary and starts it as a subprocess.
// Use Start to launch a server, then connect with the official milvus-sdk-go:
//
//	server, err := milvuslite.Start("./milvus.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Stop()
//
//	c, _ := client.NewClient(ctx, client.Config{Address: server.Addr()})
package milvuslite
