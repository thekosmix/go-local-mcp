# go-local-mcp

A lightweight, low-footprint **Model Context Protocol (MCP)** server written in Go. This suite provides a set of essential utilities (web search, webpage extraction, document reading, and secure remote code compilation) to enhance local or cloud LLMs.

---

## What Does It Do?

This MCP server exposes four tools that LLMs can invoke to interact with the web, local files, and compilers:

1. **`search`**
   * **Description:** Queries DuckDuckGo to fetch text summaries and search snippets.
   * **Input:** `query` (string)

2. **`read_local_document`**
   * **Description:** Safely reads local files (text/markdown) and automatically truncates the output if it exceeds token safety limits.
   * **Input:** `file_path` (string)

3. **`fetch_url_content`**
   * **Description:** Downloads web pages and parses out layout, styling, and scripts to extract raw readable body text.
   * **Input:** `url` (string)

4. **`code_compiler`**
   * **Description:** Sends source code to a remote PHP compiler backend, follows compilation and execution redirects, bypasses ngrok warnings, and returns the stdout/stderr result.
   * **Input:** `language` (e.g. `py`, `cpp`, `c`, `pl`, `php`, `bash`, `awk`), `code` (string), and optional `input` (stdin).
   * **Supported Languages:** `awk`, `bash`, `c`, `cpp`, `php`, `py` (Python), `pl` (Perl). It automatically normalizes full language names (e.g., `python` -> `py`, `perl` -> `pl`) to the correct backend identifiers.

---

## How to Build and Setup

### Prerequisites
* Go 1.20 or newer installed on the system.

### 1. Build the Suite
Navigate to the `go-local-mcp` directory and build the multi-file Go package:
```bash
# install dependency
go get github.com/modelcontextprotocol/go-sdk/mcp
# setup environment
go mod init go-local-mcp
go mod tidy
# Build the binary (disabling VCS stamping to prevent repository permission issues if building in sandboxed environments)
go build -o go-local-mcp 
```

### 2. Run Directly
To test or run the server immediately in the foreground:
```bash
# Run the built binary
./go-local-mcp

# Or run directly from source
go run .
```
The server will start listening for HTTP requests on port `:8080`.

---

## Running as a System Process (systemd)

To make sure the MCP server runs persistently in the background, starts automatically on system boot, and restarts if it crashes, you can configure it as a `systemd` service.

### 1. Create the Service File
Create a new file `/etc/systemd/system/go-local-mcp.service` using `sudo` and your preferred text editor:
```bash
sudo nano /etc/systemd/system/go-local-mcp.service
```

### 2. Populate Service Configuration
Add the following content (make sure to replace `droidian` and path references with your actual system username and paths):
```ini
[Unit]
Description=Go Local MCP Server Suite
After=network.target

[Service]
Type=simple
User=droidian
WorkingDirectory=/home/droidian/git/go-local-mcp
ExecStart=/home/droidian/git/go-local-mcp/go-local-mcp
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### 3. Start and Enable the Service
Reload the systemd daemon, enable the service to launch on boot, and start it immediately:
```bash
# Reload systemd configuration
sudo systemctl daemon-reload

# Enable auto-start on boot
sudo systemctl enable go-local-mcp.service

# Start the service
sudo systemctl start go-local-mcp.service
```

### 4. Manage the Service
Use standard systemd commands to check status or restart:
```bash
# Check running status and port bindings
sudo systemctl status go-local-mcp.service

# Restart the service (e.g., after modifying code)
sudo systemctl restart go-local-mcp.service

# Stop the service
sudo systemctl stop go-local-mcp.service
```

### 5. View Logs
View logs in real-time to debug request payloads and response times:
```bash
sudo journalctl -u go-local-mcp.service -f
```
