# Executioner

Executioner is a secure, high-performance code execution engine built in Go. It allows you to run untrusted code in isolated Docker containers with strict resource limits and security hardening.

## Features

- **Secure Sandbox**: Executes code in hardened Docker containers.
- **Multi-Language Support**: Support for C++, Python, JavaScript, and TypeScript out of the box.
- **High Concurrency**: Uses an asynchronous job queue and worker pool for efficient job processing.
- **Resource Management**: Strict CPU, Memory, and PID limits.
- **Security Hardened**: No networking, dropped capabilities, no-new-privileges, and memory-backed execution environments.
- **Rate Limiting**: Built-in global and per-IP rate limiting.
- **Observability**: Prometheus-compatible metrics for monitoring throughput, latency, and resource usage.
- **Auto-Pull**: Automatically pulls required Docker images on startup.

## Prerequisites

- Go (1.21 or later)
- Docker Engine
- PostgreSQL (optional — only needed when `EXECUTIONER_DB_ENABLED=true`)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/itstheanurag/executioner.git
cd executioner
```

### 2. Configure environment variables

Copy the example environment file and adjust the values as needed:

```bash
cp .env.example .env
```

### 3. Run the application

```bash
go run cmd/api/main.go
```

The server will start on port `8080` by default. Required Docker images (like `python:3.11-slim`) will be pulled automatically if they are missing.

### 4. Open the playground

Visit [http://localhost:8080](http://localhost:8080) in your browser. The built-in frontend provides a code editor, stdin panel, and live output — no separate frontend server required.

For local frontend development on a different port (e.g. `localhost:3000`), CORS is enabled via `EXECUTIONER_SERVER_CORS_ALLOWED_ORIGINS` in your `.env` file.

## API Usage

### Execute Code

**Endpoint**: `POST /execute`

**Request Body**:

```json
{
  "language": "python",
  "source_code": "print('Hello, Executioner!')",
  "time_limit": 2
}
```

**Example Curl**:

```bash
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{"language": "python", "source_code": "print(42)"}'
```

### List Languages

**Endpoint**: `GET /languages`

Returns the list of supported programming languages.

### Metrics

**Endpoint**: `GET /metrics`

Returns Prometheus-compatible metrics for monitoring the system.

## Supported Languages

| Language   | ID           | Image              |
| ---------- | ------------ | ------------------ |
| C++        | `cpp`        | `gcc:13`           |
| Python     | `python`     | `python:3.11-slim` |
| JavaScript | `javascript` | `node:20-slim`     |
| TypeScript | `typescript` | `node:20-slim`     |

## Architecture

For a detailed look at how Executioner is built, see [architecture.md](./architecture.md).

## License

MIT
