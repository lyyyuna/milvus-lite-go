package linuxamd64

import "embed"

// Lib contains the milvus-lite binary and shared libraries for linux/amd64.
//
//go:embed lib/*
var Lib embed.FS
