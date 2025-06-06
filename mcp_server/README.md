# MCP Server for Kubectl

The MCP (Minimal Control Plane) Server provides an HTTP API to execute `kubectl` commands. It allows clients to interact with a Kubernetes cluster by sending `kubectl` commands via HTTP requests and receiving the output.

This server is designed to be a lightweight intermediary for scenarios where direct `kubectl` access is not feasible or desired, but programmatic access to `kubectl` functionality is needed.

## Prerequisites

- **Go:** Version 1.21 or newer.
- **Docker:** (Optional) If you plan to build and run the server using Docker.
- **Kubectl:** Must be installed and configured to access your target Kubernetes cluster. The server itself will execute `kubectl` commands, so it relies on a working `kubectl` setup in the environment where the server (or its Docker container) runs.

## Building and Running Locally

1.  **Clone the repository (if applicable) or navigate to the `mcp_server` directory.**

2.  **Build the Go binary:**
    ```bash
    cd mcp_server
    go build
    ```
    This will create an executable file named `mcp_server` (or `mcp_server.exe` on Windows).

3.  **Run the server:**
    ```bash
    ./mcp_server
    ```
    The server will start and listen on the default port `8080`. You should see a log message indicating it's listening:
    `YYYY/MM/DD HH:MM:SS MCP Server listening on port 8080...`

## Building and Running with Docker

1.  **Navigate to the `mcp_server` directory.**
    This directory contains the `Dockerfile`.

2.  **Build the Docker image:**
    ```bash
    docker build -t mcp-server .
    ```
    This command builds a Docker image tagged `mcp-server`.

3.  **Run the Docker container:**
    ```bash
    docker run -p 8080:8080 -v ~/.kube:/root/.kube:ro mcp-server
    ```
    - `-p 8080:8080`: Maps port 8080 on your host to port 8080 in the container.
    - `-v ~/.kube:/root/.kube:ro`: Mounts your local Kubernetes configuration directory (`~/.kube`) into the container at `/root/.kube` in read-only mode. This allows `kubectl` inside the container to access your cluster. **Note:** The path `/root/.kube` is the default location where `kubectl` inside the Alpine-based container will look for its configuration. Adjust if your base image in the Dockerfile uses a different non-root user or home directory.

## API Usage

### Endpoint: `/mcp`

-   **Method:** `POST`
-   **Request Body (JSON):**
    The request body must be a JSON object specifying the command to run. Currently, the server has a strict validation for only `kubectl get pods`.

    ```json
    {
      "command": "kubectl",
      "args": ["get", "pods"]
    }
    ```
    -   `command`: Should be `"kubectl"`.
    -   `args`: An array of strings representing the arguments to `kubectl`. For now, this must be `["get", "pods"]`.

-   **Success Response (JSON):**
    If the command is processed (even if `kubectl` itself returns an error, like no resources found), the server returns an HTTP 200 OK.

    ```json
    {
      "output": "<output of the kubectl command>",
      "error": "<stderr output from kubectl, if any, or execution error message>"
    }
    ```
    -   `output`: Contains the standard output (and standard error for `kubectl` execution errors) from the `kubectl` command.
    -   `error`: Contains an error message if `kubectl` itself failed (e.g., command syntax error, non-zero exit code from `kubectl`). If the `kubectl` command ran successfully, this field might be an empty string or contain stderr output if `kubectl` wrote to it despite a zero exit code.

-   **Server Error Response (Non-200 HTTP):**
    If the server encounters an issue before it can even attempt to run `kubectl` (e.g., invalid JSON, unsupported command by the server's validation), it will return an appropriate HTTP error code (e.g., 400 Bad Request, 405 Method Not Allowed) with a plain text error message or a JSON `MCPResponse` detailing the server-side error.

    Example for a disallowed command before `kubectl` execution:
    HTTP Status: 400 Bad Request
    ```
    Currently only 'kubectl get pods' is supported
    ```

## Testing

To run the unit tests for the server:

1.  **Navigate to the `mcp_server` directory.**
2.  **Run the tests:**
    ```bash
    go test -v ./...
    ```
    This command will execute all test files (`*_test.go`) in the current directory and any subdirectories. The `-v` flag enables verbose output.

## Future Enhancements (Optional)

-   **Broader `kubectl` command support:** Allow a wider range of `kubectl` commands and arguments, potentially with a configurable allow-list or block-list.
-   **Namespace support:** Explicitly allow specifying namespaces in requests.
-   **Authentication & Authorization:** Implement proper authentication for the API endpoint.
-   **More robust error handling:** Differentiate more clearly between server errors and `kubectl` execution errors in the response.
-   **Configuration:** Allow configuration of port, logging levels, etc., via environment variables or a config file.
-   **Streaming output:** For long-running `kubectl` commands (like `logs -f`), support streaming the output.
