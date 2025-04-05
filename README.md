# WebMVCEmployees

WebMVCEmployees is a Go-based web application that manages employee data. It includes integrated Swagger documentation for the API and an admin dashboard to view tables.

## Features

- **REST API Endpoints:** Manage employees via a RESTful API.
- **Swagger Documentation:** Automatic generation of API docs using swaggo.
- **Admin Dashboard:** View database tables by logging in at `localhost:8081` (Username: `root`, Password: `root`).

## Prerequisites

- **Go 1.16+** installed.
- **Docker:** Required if you are running the MongoDB container.
- **swag:** To generate Swagger documentation.

## Quick Start: Running the Server

### Option A: Using Pre-built Executables

You can run the server using the pre-built executables:

- **For Windows (64-bit):**  
  Run `webmvc_employees.exe` by double-clicking it or from the command line.

- **For macOS (Apple Silicon/M1):**  
  Run `webmvc_employees.dmg` from the terminal:

  ```bash
  ./webmvc_employees
  ```

Option B: Using go run

Alternatively, you can start the server directly using the Go tool:

```bash
  go run cmd/webmvc_employees/main.go
```

This will compile and run the server on port **8080**.

### Accessing the Application

- **API Server:**  
  The server listens on port **8080** by default.

- **Admin Dashboard:**  
  Navigate to [http://localhost:8081](http://localhost:8081) in your browser.  
  Login using:
  - **Username:** `root`
  - **Password:** `root`

Below is a revised version of that section with improved formatting and clear instructions:

## Running Tests

To run the tests (located in `tests/employee_test.go`), use the following command from the project root:

```bash
go test ./tests/...
```

This command will execute all tests under the tests directory.

Swagger Documentation & Building the Executables

1. Install swag for Swagger Docs

Install swag using the following command:

go install github.com/swaggo/swag/cmd/swag@latest
export PATH=$PATH:$HOME/go/bin

If you are using zsh, reload your shell:

```bash

source ~/.zshrc
```

2. Generate Swagger Documentation

Run the command below from the root of the project to automatically generate your Swagger docs:

```bash

swag init -g docs/doc.go --parseDependency --parseInternal --output ./docs
```

This command creates the documentation files in the ./docs directory.

3. Building the Executables

To build the executables yourself, use the following commands from the project root:
Build for Windows (64-bit):

```bash
GOOS=windows GOARCH=amd64 go build -o webmvc_employees.exe cmd/webmvc_employees/main.go
```

Build for macOS (Apple Silicon/M1):

```bash

GOOS=darwin GOARCH=arm64 go build -o webmvc_employees cmd/webmvc_employees/main.go
```
