# WebMVCEmployees

WebMVCEmployees is a Go-based web application designed to manage employee data. It features integrated Swagger documentation for the API, an admin dashboard, and seamless Docker integration for effortless deployment.

![Docker](https://img.shields.io/badge/Docker-✓-blue?logo=docker)  
🐳 **Docker-Ready**: Run the entire stack (app + MongoDB + Mongo Express) with a single command!

---

## 🚀 Quick Start with Docker Compose

Get started in **3 simple steps** using Docker:

### 1. **Set Up Environment Variables**

Create a `.env.docker` file(already done) in your project root with these values:

```env
# MongoDB
MONGO_INITDB_ROOT_USERNAME=root
MONGO_INITDB_ROOT_PASSWORD=example

# Mongo-Express Admin UI
ME_CONFIG_MONGODB_ADMINUSERNAME=root
ME_CONFIG_MONGODB_ADMINPASSWORD=example
ME_CONFIG_MONGODB_SERVER=mongodb
ME_CONFIG_BASICAUTH_USERNAME=root
ME_CONFIG_BASICAUTH_PASSWORD=root
```

### 2. **Launch the Stack**

Run this command to spin up all services:

```bash
docker compose --env-file .env.docker up --build
```

✨ **What this starts**:

- **API Server**: `webmvc_employees_app` (Go app on port `8080`)
- **MongoDB**: `webmvc_employees_mongodb` (persistent database)
- **Admin Dashboard**: `mongo_express` (MongoDB UI on port `8081`)

### 3. **Access Services**

| Service             | URL                                    | Credentials     |
| ------------------- | -------------------------------------- | --------------- |
| **API Server**      | http://localhost:8080                  | -               |
| **Admin Dashboard** | http://localhost:8081                  | `root` / `root` |
| **MongoDB**         | `mongodb://root:example@mongodb:27017` | -               |

---

## 🛠️ Docker Management Cheatsheet

| Command                                   | Description                    |
| ----------------------------------------- | ------------------------------ |
| `docker compose up --build`               | Start/rebuild the entire stack |
| `docker compose down`                     | Stop and remove containers     |
| `docker compose logs -f`                  | Follow container logs          |
| `docker compose restart webmvc_employees` | Hot-reload the Go app          |

---

## 🔄 Alternative Workflows

### **Run Without Docker**

For local development without containers:

```bash
# 1. Start MongoDB manually (or use cloud Atlas)
# 2. Run the Go server
go run cmd/webmvc_employees/main.go
```

### **Pre-built Executables**

Download or build standalone binaries:

| Platform                  | Command                                                                                   |
| ------------------------- | ----------------------------------------------------------------------------------------- |
| **Windows (64-bit)**      | `GOOS=windows GOARCH=amd64 go build -o webmvc_employees.exe cmd/webmvc_employees/main.go` |
| **macOS (Apple Silicon)** | `GOOS=darwin GOARCH=arm64 go build -o webmvc_employees cmd/webmvc_employees/main.go`      |

---

## 📚 Swagger Documentation

Automatically generate API docs after code changes:

```bash
swag init -g docs/doc.go --parseDependency --parseInternal --output ./docs
```

Access Swagger UI at:  
**http://localhost:8080/swagger/index.html**

---

## 🧪 Testing

Run the test suite with:

```bash
go test -v ./tests/...
```

---

## 📦 Dependency Diagram

```mermaid
graph TD
  A[Go App] --> B[MongoDB]
  C[Mongo Express] --> B
  D[User] --> A
  D --> C
```
