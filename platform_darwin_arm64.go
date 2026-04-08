//go:build darwin && arm64

package milvuslite

import darwinarm64 "github.com/lyyyuna/milvus-lite-go/platform/darwin-arm64"

func init() {
	platformLib = darwinarm64.Lib
}
