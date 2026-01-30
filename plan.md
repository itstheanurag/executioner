# Judge0-Golang — MVP Implementation Plan

## 1. Objective

Build a Go-based, secure, lightweight code execution platform inspired by Judge0.

The platform will accept user-submitted source code, execute it safely in isolated environments, and return deterministic execution results such as output, errors, execution time, and memory usage.

The primary use case is competitive programming platforms and learning environments similar to LeetCode and GeeksForGeeks.

---

## 2. Explicit Non-Goals (MVP)

The following are intentionally excluded from the MVP:

- User authentication or authorization
- Web-based IDE or editor
- Code storage or submission history
- Distributed execution or autoscaling
- Advanced sandboxing (Firecracker, gVisor)
- Plagiarism detection
- Scoring, leaderboards, or contests

---

## 3. MVP Feature Set

### 3.1 Core Capabilities

- Secure execution of untrusted code
- Support for multiple programming languages
- Enforced CPU, memory, and time limits
- Capture of stdout, stderr, and exit codes
- Simple HTTP-based API
- Deterministic execution results

### 3.2 Supported Languages (Initial)

- C++ (compiled)
- Python (interpreted)
- JavaScript (Node.js)

Languages must be configurable and extendable.

---

## 4. High-Level Architecture

Request Flow:

Client  
→ API Server  
→ Job Queue  
→ Worker Pool  
→ Sandbox (Docker)  
→ Result

### Components

- API Server (Go)
- In-memory Job Queue
- Worker Pool
- Execution Engine
- Docker-based Sandbox
- Language Runtime Registry

---

## 5. Execution Flow (Step-by-Step)

1. Client sends execution request
2. API validates request payload
3. Job is pushed into the queue
4. Worker picks up the job
5. Temporary workspace is created
6. Source code is written to files
7. Sandbox container is started
8. Code is compiled (if required)
9. Code is executed with limits
10. stdout, stderr, exit code collected
11. Execution metrics recorded
12. Container is destroyed
13. Workspace is cleaned up
14. Result is returned to client

---

## 6. API Design

### 6.1 Execute Code Endpoint

Endpoint:

    POST /execute

Request Body:

    {
      "language": "cpp",
      "source_code": "#include <iostream>\nint main(){ std::cout << 42; }",
      "stdin": "",
      "time_limit": 2,
      "memory_limit": 256
    }

Response Body:

    {
      "status": "success",
      "stdout": "42",
      "stderr": "",
      "exit_code": 0,
      "time_ms": 120,
      "memory_kb": 32768
    }

---

## 7. Sandbox and Security Model

### 7.1 Isolation Strategy

- One Docker container per execution
- Non-root user inside container
- Read-only root filesystem
- No network access
- No access to host filesystem

### 7.2 Resource Limiting

- CPU limits using Docker CPU constraints
- Memory limits using Docker memory constraints
- Execution timeout enforced using:
  - Go context cancellation
  - Process termination
  - Container kill on timeout

### 7.3 Security Hardening

- Disable networking completely
- Drop all unnecessary Linux capabilities
- Use Docker default seccomp profile
- Limit number of processes
- Limit open file descriptors

---

## 8. Language Runtime Definitions

Each language runtime is defined using a configuration entry containing:

- Docker image
- Source file name
- Compile command (optional)
- Run command

Example runtimes:

C++:

- Image: gcc:13
- Source file: main.cpp
- Compile: g++ main.cpp -O2 -o main
- Run: ./main

Python:

- Image: python:3.11
- Source file: main.py
- Run: python main.py

JavaScript:

- Image: node:20
- Source file: main.js
- Run: node main.js

---

## 9. Project Structure

    judge0-golang/
    ├── cmd/
    │   └── api/
    │       └── main.go
    ├── internal/
    │   ├── api/
    │   ├── queue/
    │   ├── worker/
    │   ├── executor/
    │   ├── sandbox/
    │   ├── languages/
    │   ├── limiter/
    │   └── metrics/
    ├── configs/
    ├── scripts/
    └── plan.md

---

## 10. Job Queue Design

- Implemented using buffered Go channels
- FIFO job processing
- Fixed-size worker pool
- Backpressure applied when queue is full

Future replacements:
pool
- Backpressure applied when queue is full

Future replacements:

- Redis
- RabbitMQ
- Kafka

---

## 11. Worker Pool Design

### Responsibilities

- Fetch jobs from queue
- Execute jobs using executor
- Handle timeouts and failures
- Return execution results

### Configuration

- Fixed number of workers
- Configurable via environment variables

---

## 12. Executor Design

### Responsibilities

- Create isolated workspace
- Write source code files
- Resolve language runtime
- Invoke sandbox
- Collect outputs and metrics

---

## 13. Sandbox Implementation (Docker)

Execution lifecycle:

1. Create container from language image
2. Copy source code into container
3. Run compile command (if present)
4. Run execution command
5. Capture stdout and stderr
6. Measure execution time
7. Enforce memory and time limits
8. Destroy container after execution

---

## 14. Execution Result Model

    type ExecutionResult struct {
      Status    string
      Stdout    string
      Stderr    string
      ExitCode  int
      TimeMs    int64
      MemoryKb  int64
      ErrorType string
    }

---

## 15. Error Classification

- Compilation Error
- Runtime Error
- Time Limit Exceeded
- Memory Limit Exceeded
- Invalid Language
- Internal Server Error

---

## 16. Logging and Observability

- Structured JSON logs
- Execution start and end logs
- Error logs per execution
- No tracing in MVP

---

## 17. Performance Targets

- Container startup time under 300 ms
- Small program execution under 1 second
- Concurrent executions: 5–10
- Predictable memory usage

---

## 18. Testing Strategy

### Unit Tests

- Language registry
- Command generation
- Limit enforcement logic

### Integration Tests

- End-to-end execution
- Infinite loop handling
- Memory exhaustion handling
- Malicious code attempts

---

## 19. Deployment Strategy (MVP)

- Single Linux VM
- Docker installed
- API server and workers in same process
- Configuration via environment variables

---

## 20. Post-MVP Roadmap

- Persistent execution storage
- Judge mode with test cases
- Distributed worker nodes
- Advanced sandboxing
- Additional language support
- Authentication and rate limiting

---

## 21. MVP Success Criteria

The MVP is successful if:

- Untrusted code executes safely
- Resource limits are strictly enforced
- Execution results are deterministic
- System remains stable under abuse
- API remains simple and reliable
