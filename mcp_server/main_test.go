package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os" // Added os import
	"os/exec"
	"strings"
	"testing"
)

// execCommand from main.go is used for mocking.
// var execCommand = exec.Command // Removed redeclaration

// mockExecCommand is a helper function to mock exec.Command
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestMCPHandler_ValidRequest. When the tests are run, if
// GO_WANT_HELPER_PROCESS is set, it will execute this function.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Check what command was called
	args := os.Args
	cmd, subArgs := args[3], args[4:] // os.Args[0] is test binary, os.Args[1] is -test.run=TestHelperProcess, os.Args[2] is --

	if cmd == "kubectl" && len(subArgs) > 1 && subArgs[0] == "get" && subArgs[1] == "pods" {
		fmt.Fprint(os.Stdout, "NAME          READY   STATUS    RESTARTS   AGE\npod-1         1/1     Running   0          10m\npod-2         0/1     Pending   0          5m")
	} else {
		// Default behavior or error for unhandled commands in mock
		fmt.Fprint(os.Stderr, "mock exec: unhandled command")
		os.Exit(1)
	}
	os.Exit(0)
}

func TestMCPHandler_ValidRequest(t *testing.T) {
	// Replace exec.Command with our mock
	originalExecCommand := execCommand
	execCommand = mockExecCommand
	defer func() { execCommand = originalExecCommand }()

	reqBody := MCPRequest{
		Command: "kubectl",
		Args:    []string{"get", "pods"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "/mcp", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mcpHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Errorf("Response body: %s", rr.Body.String())
	}

	var resp MCPResponse
	err = json.NewDecoder(rr.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Could not decode response: %v", err)
	}

	expectedOutput := "NAME          READY   STATUS    RESTARTS   AGE\npod-1         1/1     Running   0          10m\npod-2         0/1     Pending   0          5m"
	if !strings.Contains(resp.Output, "pod-1") || !strings.Contains(resp.Output, "pod-2") {
		t.Errorf("handler returned unexpected body for Output: got %q want to contain %q", resp.Output, expectedOutput)
	}

	if resp.Error != "" {
		t.Errorf("handler returned unexpected error: got %q", resp.Error)
	}
}

func TestMCPHandler_InvalidCommand(t *testing.T) {
	reqBody := MCPRequest{
		Command: "kubectl",
		Args:    []string{"delete", "everything"},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "/mcp", bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mcpHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	expectedErrorMsg := "Currently only 'kubectl get pods' is supported"
	if !strings.Contains(rr.Body.String(), expectedErrorMsg) {
		t.Errorf("handler returned unexpected error message: got %q want to contain %q", rr.Body.String(), expectedErrorMsg)
	}
}

func TestMCPHandler_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"command": "kubectl", "args": ["get", "pods"`) // Missing closing brace

	req, err := http.NewRequest("POST", "/mcp", bytes.NewBuffer(invalidJSON))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mcpHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Based on the updated main.go logic for incomplete JSON
	expectedErrorMsg := "Invalid JSON: Incomplete or malformed"
	if !strings.Contains(rr.Body.String(), expectedErrorMsg) {
		t.Errorf("handler returned unexpected error message: got %q want to contain %q", rr.Body.String(), expectedErrorMsg)
	}
}

func TestMCPHandler_NonPOSTRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(mcpHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}

	expectedErrorMsg := "Only POST method is allowed"
	if !strings.Contains(rr.Body.String(), expectedErrorMsg) {
		t.Errorf("handler returned unexpected error message: got %q want to contain %q", rr.Body.String(), expectedErrorMsg)
	}
}
