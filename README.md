# ATOMOS Environment

Welcome to the ATOMOS repository (formerly V.O.I.D and The Lab). This repository contains the complete logistics simulation environment, AI configuration, and core codebase for local development.

## 🚀 Setting Up the Local Environment

The core infrastructure relies on Spanner, Kafka, and Redis emulators configured specifically for this ecosystem.

### Prerequisites
- Docker & Docker Compose
- Go 1.22+
- Node.js 20+

### Starting the Infrastructure

1. Navigate to the core monorepo directory:
   ```bash
   cd the-lab-monorepo
   ```
2. Start the backend simulation stack (Database, Cache, Event Bus):
   ```bash
   docker-compose up -d
   ```
3. (Optional) Run the Go services:
   ```bash
   cd apps/backend-go
   go build ./...
   ```

### Project Structure
- `the-lab-monorepo/apps/`: Primary web portals, native mobile apps, and Go backends.
- `the-lab-monorepo/packages/`: Shared TS types, validation rules, API clients.
- `.agents/` and `.github/`: Local AI agent skills and architectural doctrines that guide development.
