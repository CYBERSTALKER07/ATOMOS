# Project: PegasusX Architectural Fixes & System Constraints

## Architecture
PegasusX is an enterprise multi-tenant B2B FMCG distribution platform with a Go microservices backend, Spanner database, Kafka messaging, Redis caching, Next.js web portals, and native mobile clients.

This project delivers 4 critical architectural fixes across the backend, frontend, and data layers:
1. **Kafka DLQ & Spanner Failure Transparency**:
   - Panic recovery inside the retry loop in `apps/backend-go/kafka/consumer.go`.
   - 4 consecutive panics / errors route message to DLQ and execute the Spanner failure updater hook to transition the job status to `FAILED`.
   - Supplier UI (`apps/supplier-portal`) displays `FAILED` job status and error summaries.
2. **Server-Sent Events (SSE) Migration**:
   - Backend `GET /v1/supplier/events` streaming endpoint using `text/event-stream`, `X-Accel-Buffering: no`, 15-second keep-alive comments (`: ping\n\n`), and reconnection parameters (`retry: 3000`).
   - Replaces Supplier WebSocket connections with lightweight `EventSource` in `apps/supplier-portal` and `@pegasusx/ws-refresh-contract`.
3. **Frontend Geospatial Validation**:
   - Turf.js integration (`@turf/kinks`, `@turf/simplify`, `@turf/helpers`, `@turf/clean-coords`) in `packages/validation` and `packages/ui-maps`.
   - Intercepts polygon submission in Mapbox/MapLibre Draw UI (`ControlTowerCommandPanel.tsx`), auto-simplifying polygons with >50 vertices to <50 vertices, and rejecting/blocking self-intersecting (kinked) shapes with descriptive client-side errors.
4. **FCM Stale Token Race Condition Resolution**:
   - Schema enhancement to `DeviceTokens` table in `apps/backend-go/schema/spanner.ddl` adding `SessionId STRING(128)` and `DeviceId STRING(128)` with indexes.
   - Updates `purgeStaleToken` in `apps/backend-go/notifications/fcm.go` and `platform/repository.go` to explicitly match tokens to device sessions, preventing blind deletion of newly issued tokens during rapid re-installs.

---

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Kafka DLQ Retry & Panic Recovery | Per-attempt panic recovery inside retry loop up to MaxAttempts (4) | M1 | ORIGINAL_REQUEST § R1 |
| 2 | Kafka DLQ Spanner Job Status Update | Write FAILED status to Spanner job tables upon routing to DLQ | M1 | ORIGINAL_REQUEST § R1 |
| 3 | Supplier UI DLQ Failure Display | Render FAILED alerts and error summaries in Supplier Portal | M1 | ORIGINAL_REQUEST § R1 |
| 4 | Backend SSE Endpoint | GET /v1/supplier/events streaming handler, keep-alive pings & adapter | M2 | ORIGINAL_REQUEST § R2 |
| 5 | Supplier Web & Contract SSE Client | Replace WebSocket with EventSource in supplier portal & refresh contract | M2 | ORIGINAL_REQUEST § R2 |
| 6 | Supplier Native SSE Handlers | Provide SSE client stream adapters for mobile (Android/iOS) | M2 | ORIGINAL_REQUEST § R2 |
| 7 | Geospatial Turf.js Validation Logic | Auto-simplify >50 vertices to <50 vertices & detect self-intersections | M3 | ORIGINAL_REQUEST § R3 |
| 8 | Mapbox Draw UI Interception | Intercept polygon submission in ControlTowerCommandPanel.tsx | M3 | ORIGINAL_REQUEST § R3 |
| 9 | DeviceTokens Schema Session Columns | Add SessionId & DeviceId columns and indexes in Spanner DDL | M4 | ORIGINAL_REQUEST § R4 |
| 10 | Session-Aware Token Purging | Update purgeStaleToken to match specific session/device | M4 | ORIGINAL_REQUEST § R4 |
| 11 | E2E & Programmatic Verification Suite | 4 acceptance criteria verification suites (DLQ panic test, SSE curl, Turf validation, FCM session test) | M5 | ORIGINAL_REQUEST § Acceptance Criteria |

---

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Kafka DLQ UI Transparency | `apps/backend-go/kafka/consumer.go`, `apps/backend-go/kafka/dlq_spanner.go`, `apps/supplier-portal` | none | DONE |
| M2 | SSE Migration | `apps/backend-go/ws/sse.go`, `apps/backend-go/ws/handler.go`, `packages/ws-refresh-contract`, `apps/supplier-portal` | none | DONE |
| M3 | Frontend Geospatial Validation | `packages/validation`, `packages/ui-maps`, `apps/supplier-portal/components/ControlTowerCommandPanel.tsx` | none | DONE |
| M4 | FCM Stale Token Race Condition Fix | `apps/backend-go/schema/spanner.ddl`, `apps/backend-go/platform/repository.go`, `apps/backend-go/notifications/fcm.go` | none | DONE |
| M5 | E2E Integration & Verification Suite | End-to-end tests across all 4 acceptance criteria | M1, M2, M3, M4 | DONE |


---

## Interface Contracts

### 1. Kafka DLQ Hook & Spanner Status Update
- **Hook Signature**:
  ```go
  type DLQHook func(ctx context.Context, msg kafka.Message, reason error) error
  ```
- **ConsumerDeps**:
  ```go
  type ConsumerDeps struct {
      Brokers     []string
      GroupID     string
      Topic       string
      Topics      []string
      Handler     EventHandler
      DLQWriter   DLQWriter
      OnDLQ       DLQHook
      MaxAttempts int
      Auth        kafkautil.ClientAuth
  }
  ```
- **DLQ Reason Headers**:
  - `dlq_reason`: error message / panic trace
  - `original_topic`: string
  - `original_partition`: string
  - `original_offset`: string

### 2. SSE Server-Sent Events Contract
- **Endpoint**: `GET /v1/supplier/events` (and `GET /v1/events`)
- **Headers**:
  - `Content-Type: text/event-stream; charset=utf-8`
  - `Cache-Control: no-cache, no-transform`
  - `Connection: keep-alive`
  - `X-Accel-Buffering: no`
- **Stream Framing**:
  - Initial handshake: `retry: 3000\n\n: connected\n\n`
  - Periodic keep-alive ping: `: ping\n\n` (every 15 seconds)
  - Broadcast event: `id: <timestamp_or_event_id>\nevent: <eventType>\ndata: <JSON>\n\n`

### 3. Frontend Geospatial Validation Contract
- **Function**: `validateAndSimplifyPolygon(geojson: GeoJSON.Polygon | Feature<Polygon> | Coordinates, options?: ValidationOptions)`
- **Output**:
  ```typescript
  export interface PolygonValidationResult {
    valid: boolean;
    error?: string;
    hasKinks?: boolean;
    vertexCount: number;
    simplified: boolean;
    geojson?: GeoJSON.Polygon;
  }
  ```
- **Thresholds**: Max vertices = 50 (auto-simplifies if >50). Kinks / self-intersections strictly rejected.

### 4. FCM Device Token Session Contract
- **Spanner Schema**:
  ```sql
  CREATE TABLE DeviceTokens (
    Token          STRING(512) NOT NULL,
    ActorId        STRING(36)  NOT NULL,
    ActorRole      STRING(20)  NOT NULL,
    Platform       STRING(20)  NOT NULL,
    DeviceId       STRING(128),
    SessionId      STRING(128),
    UpdatedAt      TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
  ) PRIMARY KEY (Token);
  ```
- **Purge Method**:
  ```go
  func (r *DeviceTokenRepository) PurgeStaleToken(ctx context.Context, token string, sessionID string) error
  func (r *DeviceTokenRepository) DeleteTokenBySession(ctx context.Context, actorID string, sessionID string) error
  ```

---

## Code Layout
- `apps/backend-go/kafka/`: Kafka consumer, DLQ writer, retry & panic loop, Spanner failure updater
- `apps/backend-go/ws/`: WebSocket hub and Server-Sent Events (SSE) adapter & handlers
- `apps/backend-go/notifications/`: FCM notification sender & stale token purger
- `apps/backend-go/platform/`: Device token repository & HTTP endpoints
- `apps/backend-go/schema/spanner.ddl`: Canonical Spanner DDL definitions
- `packages/validation/`: Polygon validation, Turf.js rules, Zod schemas
- `packages/ui-maps/`: Mapbox/MapLibre map components & draw editors
- `packages/ws-refresh-contract/`: Real-time SSE / event definitions
- `apps/supplier-portal/`: Next.js portal with SSE client hooks, DLQ failure alerts, and Mapbox draw controls
