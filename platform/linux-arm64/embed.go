package linuxarm64

import "embed"

// Lib contains the milvus-lite binary and shared libraries for linux/arm64.
//
//go:embed lib/*
var Lib embed.FS
