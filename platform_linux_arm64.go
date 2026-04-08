//go:build linux && arm64

package milvuslite

import linuxarm64 "github.com/lyyyuna/milvus-lite-go/platform/linux-arm64"

func init() {
	platformLib = linuxarm64.Lib
}
