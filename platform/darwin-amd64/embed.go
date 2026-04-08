package darwinamd64

import "embed"

// Lib contains the milvus-lite binary and shared libraries for darwin/amd64.
//
//go:embed lib/*
var Lib embed.FS
