# Exchange Backend

A scalable microservices backend for a trading exchange platform built with Go, gRPC, Docker, and Kubernetes.

## 📐 Architecture

The project follows a **Microservices Architecture** with the **Database-per-Service** pattern and **JWT-based authentication**.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client (HTTP)                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Kong API Gateway (:8000)                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  Plugins: CORS │ Rate Limiting │ Correlation ID │ JWT (protected)   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
          │                                              │
          │ (No JWT required)                            │ (JWT Required)
          ▼                                              ▼
┌──────────────────────────┐                ┌──────────────────────────┐
│     Auth Service         │                │    Orders Service        │
│  ┌────────────────────┐  │                │  ┌────────────────────┐  │
│  │  HTTP (:8080)      │  │                │  │  HTTP (:8081)      │  │
│  │  grpc-gateway      │  │                │  │                    │  │
│  └────────┬───────────┘  │                │  └────────────────────┘  │
│           │              │                │                          │
│  ┌────────▼───────────┐  │                │                          │
│  │  gRPC (:9090)      │  │                │                          │
│  │  AuthService       │  │                │                          │
│  └────────────────────┘  │                │                          │
└───────────┬──────────────┘                └────────────┬─────────────┘
            │                                            │
            ▼                                            ▼
┌──────────────────────────┐                ┌──────────────────────────┐
│   PostgreSQL (Auth)      │                │  PostgreSQL (Orders)     │
│   - users                │                │   - orders               │
│   - refresh_tokens       │                │                          │
└──────────────────────────┘                └──────────────────────────┘
```

### Key Components

| Component | Description |
|-----------|-------------|
| **Kong Gateway** | Single entry point (API Gateway) handling routing, CORS, rate limiting, and JWT verification for protected routes |
| **Auth Service** | Handles user authentication via gRPC with HTTP gateway. Issues JWT access tokens and refresh tokens |
| **Orders Service** | Handles order processing. Protected by JWT authentication at Kong level |
| **PostgreSQL** | Each service has its own dedicated database (database-per-service pattern) |

## 🔐 Authentication Flow

```
┌────────┐                    ┌──────┐                    ┌──────────┐
│ Client │                    │ Kong │                    │   Auth   │
└───┬────┘                    └──┬───┘                    └────┬─────┘
    │                            │                              │
    │  POST /auth/signup         │                              │
    │───────────────────────────>│  POST /auth/signup           │
    │                            │─────────────────────────────>│
    │                            │      { user_id, message }    │
    │      { user_id, message }  │<─────────────────────────────│
    │<───────────────────────────│                              │
    │                            │                              │
    │  POST /auth/login          │                              │
    │───────────────────────────>│  POST /auth/login            │
    │                            │─────────────────────────────>│
    │                            │  { access_token,             │
    │  { access_token,           │    refresh_token,            │
    │    refresh_token,          │    expires_in }              │
    │    expires_in }            │<─────────────────────────────│
    │<───────────────────────────│                              │
    │                            │                              │
    │  GET /orders               │                              │
    │  Authorization: Bearer xxx │                              │
    │───────────────────────────>│                              │
    │                            │  JWT Validation              │
    │                            │  (Kong JWT Plugin)           │
    │                            │───────┐                      │
    │                            │       │ Valid?               │
    │                            │<──────┘                      │
    │                            │                              │
    │                            │  Forward to Orders Service   │
    │      { orders... }         │                              │
    │<───────────────────────────│                              │
```

## 🔌 API Endpoints

### Auth Service (Public - No JWT Required)

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| POST | `/auth/signup` | Register a new user | `{ "username": "...", "email": "...", "password": "..." }` | `{ "user_id": 1, "message": "..." }` |
| POST | `/auth/login` | Login and get tokens | `{ "email": "...", "password": "..." }` | `{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }` |
| POST | `/auth/refresh` | Refresh tokens (rotation) | `{ "refresh_token": "..." }` | `{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }` |

### Orders Service (Protected - JWT Required)

| Method | Endpoint | Description | Headers |
|--------|----------|-------------|---------|
| * | `/orders/*` | Order operations | `Authorization: Bearer <access_token>` |

## 📁 Project Structure

```
exchange-backend/
├── proto/                       # Protocol Buffers Definitions
│   ├── Maqbool/
│   │   └── auth.proto           # Auth service proto with HTTP annotations
│   └── auth/                    # Generated Go code
│       ├── auth.pb.go           # Protobuf messages
│       ├── auth_grpc.pb.go      # gRPC server/client
│       └── auth.pb.gw.go        # grpc-gateway HTTP handlers
│
├── services/                    # Microservices Source Code
│   ├── auth/                    # Auth Service (Go + gRPC)
│   │   ├── cmd/auth/main.go     # Entry point (gRPC + HTTP servers)
│   │   ├── internal/
│   │   │   ├── db/              # Database connection
│   │   │   ├── jwt/             # JWT token management
│   │   │   └── service/         # gRPC service implementation
│   │   ├── migrations/          # SQL migrations
│   │   ├── Dockerfile
│   │   └── Dockerfile.dev
│   └── orders/                  # Orders Service (Go)
│
├── k8s/                         # Kubernetes Configuration (Kustomize)
│   ├── base/                    # Common resources
│   ├── infrastructure/
│   │   ├── kong/                # API Gateway with JWT plugin
│   │   ├── postgres-auth/       # Dedicated DB for Auth
│   │   └── postgres-orders/     # Dedicated DB for Orders
│   ├── services/
│   │   ├── auth/
│   │   └── orders/
│   └── overlays/
│       ├── dev/
│       └── prod/
│
├── deploy/                      # Local Development
│   └── docker/
│       ├── docker-compose.yml   # Docker Compose setup
│       └── kong/
│           └── kong.yml         # Kong declarative config
│
├── go.mod                       # Go modules
├── skaffold.yaml                # Skaffold orchestration
└── README.md
```

## 🚀 Quick Start

### Prerequisites
- **Docker Desktop** (with Kubernetes enabled) or **Minikube**
- **Skaffold** (for the best dev experience)
- **Go 1.24+** (for local development)
- **protoc** with plugins (for regenerating proto files)

### Option 1: Docker Compose (Quickest)

```bash
cd deploy/docker
docker-compose up --build
```

Access the services at:
- **API Gateway**: `http://localhost:8000`
- **Kong Admin**: `http://localhost:8001`

### Option 2: Kubernetes Development (Production-like)

```bash
skaffold dev
```

Access the services at:
- **API Gateway**: `http://localhost:9000`

## 🧪 Testing the Auth API

### 1. Sign Up
```bash
curl -X POST http://localhost:8000/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "user_id": 1,
  "message": "User created successfully"
}
```

### 2. Login
```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a1b2c3d4e5f6...",
  "expires_in": 900
}
```

### 3. Access Protected Route
```bash
curl http://localhost:8000/orders \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 4. Refresh Token
```bash
curl -X POST http://localhost:8000/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "a1b2c3d4e5f6..."
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...(new)",
  "refresh_token": "x1y2z3...(new rotated token)",
  "expires_in": 900
}
```

## 🔧 Configuration

### Environment Variables

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `PORT` | Auth | `8080` | HTTP server port |
| `GRPC_PORT` | Auth | `9090` | gRPC server port |
| `DATABASE_URL` | Auth/Orders | - | PostgreSQL connection string |
| `JWT_SECRET` | Auth/Kong | `dev-secret-change-in-production` | Secret for JWT signing/verification |

### JWT Configuration

| Setting | Value |
|---------|-------|
| Algorithm | HS256 |
| Access Token Expiry | 15 minutes |
| Refresh Token Expiry | 7 days |
| Issuer (`iss` claim) | `exchange-auth-service` |

### Kong JWT Plugin

Kong validates JWTs on protected routes using:
- **Key Claim**: `iss` (issuer)
- **Secret**: Shared with Auth service via `JWT_SECRET`
- **Algorithm**: HS256

## 🗄️ Database Schema

### Auth Service

```sql
-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- refresh_tokens table (for token rotation)
CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 🔄 Regenerating Proto Files

If you modify `proto/Maqbool/auth.proto`, regenerate the Go code:

```bash
# Install protoc plugins (if not installed)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# Add to PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Generate
protoc \
  --proto_path=proto/Maqbool \
  --proto_path=$(go env GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis \
  --go_out=proto/auth --go_opt=paths=source_relative \
  --go-grpc_out=proto/auth --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=proto/auth --grpc-gateway_opt=paths=source_relative \
  auth.proto
```

## 🛠 Adding a New Service

1. **Scaffold**: Copy structure from `services/auth`
2. **Proto** (if gRPC): Add proto file to `proto/` and generate code
3. **Manifests**: Create `k8s/services/new-service`
4. **Database**: If needed, create `k8s/infrastructure/postgres-new-service`
5. **Wire up**:
   - Add to `k8s/overlays/dev/kustomization.yaml`
   - Add route to `k8s/infrastructure/kong/kong-config.yaml`
   - Add JWT plugin config if route needs protection
   - Add build config to `skaffold.yaml`

## 🐛 Troubleshooting

### DNS Resolution Issues in Docker

If you see `server misbehaving` errors:

```bash
# Clean up everything
cd deploy/docker
docker-compose down -v --remove-orphans
docker network prune -f

# Restart Docker (if needed)
sudo systemctl restart docker

# Rebuild
docker-compose up --build
```

### Kong JWT Errors

- Ensure `JWT_SECRET` matches between Auth service and Kong config
- Check the `iss` claim in tokens matches Kong's `key` in `jwt_secrets`
- Verify token hasn't expired

## 📝 License

MIT
