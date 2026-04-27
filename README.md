# Foehn ☁️

**Foehn** is a lightweight, **self-hostable Platform-as-a-Service** (PaaS) designed to simplify the deployment of applications directly 
from Git repositories. It automates **source code cloning**, **container builds**, **port management**, and **reverse proxy routing**, 
providing a seamless automated deployment experience on **local machine** or **private server**.

## Architecture

### Control Plane
- **foehnd**: A background daemon written in Go. It acts as the orchestrator, exposing a REST API to manage the lifecycle of applications.

### Data plane
- **Builder**: Powered by Docker, responsible for building images from source code and managing isolated application containers.
- **Proxy**: Powered by Caddy Server, responsible for dynamic routing and reverse proxying without requiring server restarts.

## Requirements

- [Go](https://go.dev/doc/install) 1.22+
- Docker installed and running
- Access to the Docker socket (`/var/run/docker.sock`)
- Caddy Server installed and running with API listening on port 2019 

## Getting Started

### Start the Data Plane
```bash
# Start Caddy with the base configuration file in the project
caddy run --config Caddyfile
```

### Start the Daemon
```bash
# Start the foehn daemon
# It listens for incoming HTTP requests on local port 4805
go run cmd/foehnd/main.go
```

## REST API Reference

### Deploy an application
- **Endpoint**: `POST /api/v1/apps`
- **Header**: `Content-Type: application/json`
#### **Request body**:
```json
{
  "url": "https://github.com/username/repository.git"
}
```
#### **Success Response** (201 Created):
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "url": "app-123e4567-e89b-12d3-a456-426614174000.foehn.localhost"
}
```

### Destroy an application
- **Endpoint**: `DELETE /api/v1/apps/{id}`
- **Path param**: `id` (App ID returned during deployment)

#### Success Response (204 No Content):
```
No body is returned
```

## License

This project is licensed under the **Apache License 2.0**.