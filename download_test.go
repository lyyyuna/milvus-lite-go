package milvuslite

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlatformTag(t *testing.T) {
	tag, err := platformTag()
	if err != nil {
		t.Fatalf("platformTag: %v", err)
	}

	expected := map[string]string{
		"darwin/arm64": "macosx_11_0_arm64",
		"darwin/amd64": "macosx_10_9_x86_64",
		"linux/amd64":  "manylinux2014_x86_64",
		"linux/arm64":  "manylinux2014_aarch64",
	}

	key := runtime.GOOS + "/" + runtime.GOARCH
	if want, ok := expected[key]; ok && tag != want {
		t.Errorf("platformTag() = %q, want %q", tag, want)
	}
}

func TestWheelFileName(t *testing.T) {
	name := wheelFileName("2.5.1", "macosx_11_0_arm64")
	if name != "milvus_lite-2.5.1-py3-none-macosx_11_0_arm64.whl" {
		t.Errorf("unexpected filename: %s", name)
	}
}

func TestParsePipIndexURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "tuna mirror",
			content: `[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple
`,
			want: "https://pypi.tuna.tsinghua.edu.cn/simple",
		},
		{
			name: "with extra spaces",
			content: `[global]
  index-url =   https://mirrors.aliyun.com/pypi/simple/
`,
			want: "https://mirrors.aliyun.com/pypi/simple/",
		},
		{
			name:    "no index-url",
			content: "[global]\ntimeout = 60\n",
			want:    "",
		},
		{
			name:    "empty",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePipIndexURL(tt.content)
			if got != tt.want {
				t.Errorf("parsePipIndexURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveWheelURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}

	plat, err := platformTag()
	if err != nil {
		t.Skip("unsupported platform")
	}

	whlName := wheelFileName("2.5.1", plat)
	url, hash, err := resolveWheelURL(defaultPyPISimpleURL, whlName)
	if err != nil {
		t.Fatalf("resolveWheelURL: %v", err)
	}

	if url == "" {
		t.Error("download URL is empty")
	}
	if hash == "" {
		t.Error("sha256 hash is empty")
	}

	t.Logf("URL: %s", url)
	t.Logf("SHA256: %s", hash)
}

func TestEnsureBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping download test in short mode")
	}

	lib, err := ensureBinary(Version)
	if err != nil {
		t.Fatalf("ensureBinary: %v", err)
	}

	// Check binary exists
	bin := filepath.Join(lib, "milvus")
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("milvus binary not found: %v", err)
	}

	// Check it's executable
	if info.Mode()&0o111 == 0 {
		t.Error("milvus binary is not executable")
	}

	t.Logf("binary at: %s (%d bytes)", bin, info.Size())
}
