//go:build linux && amd64

package milvuslite

import linuxamd64 "github.com/lyyyuna/milvus-lite-go/platform/linux-amd64"

func init() {
	platformLib = linuxamd64.Lib
}
