//go:build darwin && amd64

package milvuslite

import darwinamd64 "github.com/lyyyuna/milvus-lite-go/platform/darwin-amd64"

func init() {
	platformLib = darwinamd64.Lib
}
