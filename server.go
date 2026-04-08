package milvuslite

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Server represents a running milvus-lite instance.
type Server struct {
	cmd     *exec.Cmd
	addr    string
	libPath string
	dbFile  string
}

// Start downloads the milvus-lite binary (if needed) and starts a server.
// dbFile is the path to the local database file (e.g., "./milvus.db").
// The server listens on a random available port on localhost.
func Start(dbFile string) (*Server, error) {
	return StartWithAddress(dbFile, "")
}

// StartWithAddress starts a milvus-lite server on the specified address.
// If addr is empty, a random available port on localhost is used.
func StartWithAddress(dbFile, addr string) (*Server, error) {
	lib, err := ensureBinary(Version)
	if err != nil {
		return nil, fmt.Errorf("ensure binary: %w", err)
	}

	if addr == "" {
		port, err := freePort()
		if err != nil {
			return nil, fmt.Errorf("find free port: %w", err)
		}
		addr = fmt.Sprintf("localhost:%d", port)
	}

	absDB, err := filepath.Abs(dbFile)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	dbDir := filepath.Dir(absDB)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	bin := filepath.Join(lib, "milvus")
	cmd := exec.Command(bin, absDB, addr, "ERROR")
	cmd.Env = buildEnv(lib)
	cmd.Dir = dbDir

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start milvus: %w", err)
	}

	s := &Server{
		cmd:     cmd,
		addr:    addr,
		libPath: lib,
		dbFile:  absDB,
	}

	// Wait briefly to check if the process started successfully.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return nil, fmt.Errorf("milvus exited immediately: %w", err)
	case <-time.After(500 * time.Millisecond):
		// Process still running, good.
	}

	// Wait for gRPC port to be ready.
	if err := waitForPort(addr, 10*time.Second); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("milvus failed to become ready: %w", err)
	}

	return s, nil
}

// Addr returns the gRPC address of the running server (e.g., "localhost:19530").
func (s *Server) Addr() string {
	return s.addr
}

// Stop gracefully stops the milvus-lite server.
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	if err := s.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill milvus: %w", err)
	}

	s.cmd.Wait()
	return nil
}

// buildEnv returns the environment variables for the milvus process,
// setting the library path appropriately.
func buildEnv(lib string) []string {
	env := os.Environ()

	switch runtime.GOOS {
	case "darwin":
		env = append(env, fmt.Sprintf("DYLD_LIBRARY_PATH=%s:%s", lib, os.Getenv("DYLD_LIBRARY_PATH")))
	case "linux":
		env = append(env, fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", lib, os.Getenv("LD_LIBRARY_PATH")))
	}

	return env
}

// freePort finds an available TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForPort waits until the given address accepts TCP connections.
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}
