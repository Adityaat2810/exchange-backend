# Exchange Backend

A scalable microservices backend for a trading exchange platform built with Go, Docker, and Kubernetes.

## 📐 Architecture

The project follows a **Microservices Architecture** with the **Database-per-Service** pattern.

```mermaid
graph TB
    Client[Client / Frontend] --> Kong[Kong API Gateway :8000]
    
    subgraph "Kubernetes Generic Cluster"
        Kong -->|/auth| Auth[Auth Service]
        Kong -->|/orders| Orders[Orders Service]
        
        Auth --> AuthDB[(Auth Postgres)]
        Orders --> OrdersDB[(Orders Postgres)]
    end
```

### Key Components
- **Kong Gateway**: Acts as the single entry point (API Gateway), handling routing, correlation IDs, and CORS.
- **Auth Service**: Manages user authentication and issues JWTs. Has its own dedicated Postgres database.
- **Orders Service**: Handles order processing. Has its own dedicated Postgres database.
- **Skaffold**: Orchestrates increased development velocity with hot-reloading.

## 📁 Project Structure

```
exchange-backend/
├── services/                    # Microservices Source Code
│   ├── auth/                    # Auth Service (Go)
│   └── orders/                  # Orders Service (Go)
│
├── k8s/                         # Kubernetes Configuration (Kustomize)
│   ├── base/                    # Common resources (Permissions, Namespaces)
│   ├── infrastructure/          # Shared Platform Infrastructure
│   │   ├── kong/                # API Gateway Configuration
│   │   ├── postgres-auth/       # Dedicated DB for Auth
│   │   ├── postgres-orders/     # Dedicated DB for Orders
│   └── services/                # Application Manifests
│   │   ├── auth/
│   │   └── orders/
│   └── overlays/                # Environment Specifics
│       ├── dev/                 # Local Development (Hot reload patches)
│       └── prod/                # Production (High availability)
│
├── deploy/                      # Local tooling
│   └── docker/                  # Docker Compose fallback
│
├── skaffold.yaml                # Main Development Orchestration
└── Makefile                     # Handy commands
```

## 🚀 Quick Start

### Prerequisites
- **Docker Desktop** (with Kubernetes enabled) or **Minikube**
- **Skaffold** (for the best dev experience)
- **Go 1.21+** (optional, for local non-container executuon)

### Option 1: Kubernetes Development (Recommended)

We use **Skaffold** to watch your files, rebuild images (using `air` for hot-reload), and redeploy changes instantly.

1. **Start the Dev Environment**:
   ```bash
   skaffold dev
   ```
   *This will build images, deploy Postgres instances, setup Kong, and start your apps.*

2. **Access the Services**:
   - **API Gateway**: `http://localhost:9000`
   - **Auth API**: `http://localhost:9000/auth`
   - **Orders API**: `http://localhost:9000/orders`

### Option 2: Docker Compose (Legacy/Simple)

If you don't want to use Kubernetes locally:

```bash
cd deploy/docker
docker compose up --build
```
*Note: This might not reflect the exact production topology as closely as the K8s setup.*

## 🔧 Configuration

### API Gateway Routes
Routing is defined in `k8s/infrastructure/kong/kong-config.yaml`.
- `/auth` → `auth-service:8080`
- `/orders` → `orders-service:8081`

### Database
Each service connects to its own isolated database schema/instance.
- **Auth**: `postgres://user:password@postgres-auth:5432/auth_db`
- **Orders**: `postgres://user:password@postgres-orders:5432/orders_db`

## 🛠 Adding a New Service

1.  **Scaffold**: Copy structure from `services/auth`.
2.  **Manifests**: Create `k8s/services/new-service`.
3.  **Database**: If needed, create `k8s/infrastructure/postgres-new-service`.
4.  **Wire up**:
    - Add to `k8s/overlays/dev/kustomization.yaml`.
    - Add route to `k8s/infrastructure/kong/kong-config.yaml`.
    - Add build config to `skaffold.yaml`.

## 📝 License
MIT
