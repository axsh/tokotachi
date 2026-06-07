#!/bin/bash
set -e

echo "Running Unit Tests..."
go test ./internal/...

echo "Building Binary..."
mkdir -p bin
go build -o bin/function.exe cmd/main.go
