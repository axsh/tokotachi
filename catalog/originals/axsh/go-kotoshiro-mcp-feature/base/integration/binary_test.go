package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinary(t *testing.T) {
	// Locate the binary assumed to be built in ../bin/function.exe
	// This test runs from templates/go-standard/integration/
	binaryPath := filepath.Join("..", "bin", "function.exe")

	// 1. Check if binary exists (implicit in exec, but good to be explicit or let exec fail)

	// 2. Run with --help to verify it starts up as a Kuniumi app
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run binary with --help: %v\nOutput: %s", err, output)
	}

	outStr := string(output)
	if !strings.Contains(outStr, "TemplateFunc") {
		t.Errorf("Expected output to contain app name 'TemplateFunc', got: %s", outStr)
	}

	// 3. Run the 'add' function via CGI mode
	cmdCgi := exec.Command(binaryPath, "cgi")
	// Append PATH_INFO to current environment
	cmdCgi.Env = append(os.Environ(), "PATH_INFO=/Add")

	// Input JSON
	input := `{"x": 10, "y": 20}`
	cmdCgi.Stdin = strings.NewReader(input)

	outputCgi, err := cmdCgi.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run binary cgi: %v\nOutput: %s", err, outputCgi)
	}

	outStrCgi := string(outputCgi)
	// Output should look like HTTP response (CGI)
	// We check for 200 OK and result

	if !strings.Contains(outStrCgi, "Status: 200 OK") {
		t.Errorf("Expected 200 OK, got: %s", outStrCgi)
	}
	if !strings.Contains(outStrCgi, "30") {
		t.Errorf("Expected result 30, got: %s", outStrCgi)
	}
}
