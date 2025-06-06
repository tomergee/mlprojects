package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

// MCPRequest defines the structure for incoming requests
type MCPRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// MCPResponse defines the structure for outgoing responses
type MCPResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func mcpHandler(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	log.Printf("Received request: Method=%s, URL=%s, RemoteAddr=%s", r.Method, r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("Error: Invalid method %s", r.Method)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Log request body (or at least its attempt)
	var req MCPRequest
	// Create a TeeReader to log the request body
	var requestBody bytes.Buffer
	teeReader := http.MaxBytesReader(w, r.Body, 1048576) // Limit request size to 1MB

	decoder := json.NewDecoder(io.TeeReader(teeReader, &requestBody))
	err := decoder.Decode(&req)

	log.Printf("Request body: %s", requestBody.String())

	if err != nil {
		detailedError := "Invalid request body"
		if ute, ok := err.(*json.UnsupportedTypeError); ok {
			detailedError = fmt.Sprintf("Unsupported type: %v", ute)
			log.Printf("Error decoding request body: %s", detailedError)
			http.Error(w, detailedError, http.StatusBadRequest)
		} else if se, ok := err.(*json.SyntaxError); ok {
			detailedError = fmt.Sprintf("Syntax error at offset %d: %s", se.Offset, se.Error())
			log.Printf("Error decoding request body: %s", detailedError)
			http.Error(w, detailedError, http.StatusBadRequest)
		} else if err == io.ErrUnexpectedEOF || err == io.EOF {
			detailedError = fmt.Sprintf("Invalid JSON: Incomplete or malformed: %v", err)
			log.Printf("Error decoding request body: %s", detailedError)
			http.Error(w, detailedError, http.StatusBadRequest)
		} else {
			log.Printf("Error decoding request body: %v", err)
			http.Error(w, detailedError, http.StatusBadRequest) // Default "Invalid request body"
		}
		return
	}

	log.Printf("Parsed request: Command=%s, Args=%v", req.Command, req.Args)

	// Basic command validation (can be expanded)
	if req.Command == "" {
		log.Printf("Error: Empty command in request")
		http.Error(w, "Command cannot be empty", http.StatusBadRequest)
		return
	}

	// For now, we only support "kubectl get pods"
	// This will be expanded later
	if req.Command != "kubectl" || len(req.Args) == 0 || req.Args[0] != "get" || (len(req.Args) > 1 && req.Args[1] != "pods") {
		errMsg := "Currently only 'kubectl get pods' is supported"
		log.Printf("Error: %s. Received: %s %v", errMsg, req.Command, req.Args)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	log.Printf("Executing command: %s %v", req.Command, strings.Join(req.Args, " "))
	cmd := execCommand(req.Command, req.Args...) // Use the execCommand variable

	output, err := cmd.CombinedOutput() // Captures both stdout and stderr

	resp := MCPResponse{
		Output: string(output),
	}

	if err != nil {
		// If CombinedOutput returns an error, it means the command failed to execute or returned a non-zero exit code.
		// The 'output' variable will contain stderr in this case (or a mix of stdout and stderr).
		resp.Error = err.Error() // The error from CombinedOutput often includes stderr
		log.Printf("Error executing command '%s %s': %v. Output: %s", req.Command, strings.Join(req.Args, " "), err, string(output))
		// We still return a 200 OK with the error in the JSON body,
		// as the command execution itself (calling exec.Command) didn't fail at the HTTP layer.
		// The client should check the 'error' field in the JSON response.
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		// This is a server-side error (failed to serialize response)
		log.Printf("Critical: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully sent response for command: %s %v", req.Command, req.Args)
}

// execCommand allows mocking for testing. It's initialized to exec.Command.
var execCommand = exec.Command

func main() {
	http.HandleFunc("/mcp", mcpHandler)

	port := "8080" // Make this configurable later if needed
	log.Printf("MCP Server listening on port %s...", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("FATAL: Failed to start server: %v", err)
	}
}
