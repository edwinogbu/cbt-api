//go:build !windows

// This binary only does anything on Windows (it wraps cbt-api-server.exe
// as a native Windows service via golang.org/x/sys/windows/svc). A stub
// here, rather than omitting the file, keeps `go build ./...`/`go vet
// ./...` working on every other platform (this repo's own Linux/Docker
// deployment path included) instead of failing on this one directory.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("cbt-windows-service is a Windows-only binary (wraps cbt-api-server.exe as a Windows service). Build it with GOOS=windows.")
	os.Exit(1)
}
