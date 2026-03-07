# integration-test — devctl Integration Test Suite

Python-based integration tests that verify `devctl` CLI functionality in real Docker environments.

## Overview

This feature provides the test infrastructure for validating that `devctl` subcommands (`up`, `down`, `status`, etc.) work correctly with actual Docker containers.

## Structure

- `.devcontainer/` — Development container definition (Python 3.12 + Docker CLI)
- `requirements.txt` — Python dependencies (pytest, pytest-timeout)

## Test Location

Test code is located at `$PROJECT_ROOT/tests/integration-test/` to align with the `integration_test.sh` category discovery mechanism.

## Prerequisites

- Docker Desktop running
- `devctl` binary built (`./scripts/process/build.sh`)
- Python 3.12+ with pytest installed

## Running Tests

```bash
# Build devctl first
./scripts/process/build.sh

# Run all integration tests
./scripts/process/integration_test.sh

# Run only this category
./scripts/process/integration_test.sh --categories "integration-test"

# Run a specific test
./scripts/process/integration_test.sh --categories "integration-test" --specify "test_devctl_up"
```
