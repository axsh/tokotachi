#!/bin/bash
set -e
echo "Running Integration Tests (Binary Verification)..."
go test -v ./integration/...
