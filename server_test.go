package milvuslite

import (
	"os"
	"testing"
)

func TestStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dbFile := t.TempDir() + "/test.db"

	server, err := Start(dbFile)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer server.Stop()

	if server.Addr() == "" {
		t.Error("addr is empty")
	}

	t.Logf("server running at %s", server.Addr())

	// Verify the db file directory was created
	if _, err := os.Stat(dbFile); err != nil {
		// db file may not exist until first write, but dir should
		dir := dbFile[:len(dbFile)-len("/test.db")]
		if _, err := os.Stat(dir); err != nil {
			t.Errorf("db dir not accessible: %v", err)
		}
	}

	// Stop should succeed
	if err := server.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
}

func TestStartWithAddress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dbFile := t.TempDir() + "/test.db"

	server, err := StartWithAddress(dbFile, "localhost:19530")
	if err != nil {
		t.Fatalf("StartWithAddress: %v", err)
	}
	defer server.Stop()

	if server.Addr() != "localhost:19530" {
		t.Errorf("addr = %q, want localhost:19530", server.Addr())
	}
}
