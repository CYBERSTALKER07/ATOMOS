# LEVIATHAN RC1: E2E INTEGRATION TEST PROTOCOL

**Status**: Phase 5 Complete. Monorepo Locked. Zero-Trust logistics and financial engine standing by.

## THE BOOT SEQUENCE
Open four separate terminal instances at the root of the monorepo (`the-lab-monorepo`) and execute the ignition sequence.

### Terminal 1: Local Infrastructure (Emulators)
```bash
docker-compose up -d
```

### Terminal 2: The Go Monolith (Backend)
```bash
cd apps/backend-go
export REDIS_ADDRESS=localhost:6379
export SPANNER_EMULATOR_HOST=localhost:9010
export KAFKA_BROKER_ADDRESS=localhost:9092
# (Optional) Route Global Pay traffic through the Domino simulator:
# source ../../.env.domino
go run main.go
```

### Terminal 3: Next.js Admin & Supplier Portals
```bash
cd apps/admin-portal
export NEXT_PUBLIC_API_URL=http://localhost:8080
export NEXT_PUBLIC_WS_URL=ws://localhost:8080
npm run dev
```

### Terminal 4: Expo Payload Terminal
```bash
cd apps/payload-terminal
export EXPO_PUBLIC_WS_URL=ws://localhost:8080
npx expo start
```

---

## THE DOMINO PROTOCOL (Global Pay Simulator)

The "Domino" simulator is a WireMock container that replaces all Global Pay API endpoints with deterministic local stubs. It enables testing of the 95/5 split, 3DS redirect, timeouts, and bank errors without live credentials.

### Boot
```bash
docker-compose up -d globalpay-mock
./scripts/domino-smoke.sh          # Verify 14 mappings loaded + all endpoints respond
source .env.domino                 # Point backend at the simulator
```

### Chaos Markers
Embed these strings in the `externalId` field of a payment request to trigger specific failure modes:

| Marker | Effect | HTTP Status |
|--------|--------|-------------|
| `CHAOS_DECLINE` | Card declined by issuing bank | 402 |
| `CHAOS_500` | Simulated bank processing error | 500 |
| `CHAOS_TIMEOUT` | 35-second delay (exceeds 30s client timeout) | 200 (never arrives) |
| `CHAOS_3DS` | Returns `securityCheckUrl` for 3DS redirect testing | 200 |
| `CHAOS_PENDING` | Status lookup returns `paid: false` | 200 |

Negative `recipients[].amount` in any init request triggers a **400 INVALID_SPLIT_AMOUNT** response.

### Verification Targets
1. **Happy Path**: Retailer checkout with `payment_gateway: "GLOBAL_PAY"` → session created → mock returns SUCCESS → settlement completes.
2. **95/5 Split**: Verify `recipients` array in the WireMock request log (`curl localhost:8082/__admin/requests`) contains correct supplier/platform tiyin amounts.
3. **Timeout Resilience**: Set `externalId` to `"CHAOS_TIMEOUT"` → verify backend returns 504 or retries via outbox relay.
4. **3DS Detection**: Set `externalId` to `"CHAOS_3DS"` → verify `securityCheckUrl` is propagated to the client redirect.
5. **Trace Propagation**: Inspect `X-Trace-Id` in WireMock request log → confirm it matches the originating request's trace.

---

## THE ASSAULT MATRIX (Execution Targets)

1. **The Telemetry Test:** 
   Start the native driver app (Android via Android Studio or iOS via Xcode). Grant location permissions. Open `http://localhost:3000/admin/fleet` in your browser. Verify the green marker tracks your physical movement.

2. **The Desert Protocol Test:** 
   Turn off your phone's Wi-Fi and Cellular data. Complete a delivery in the native driver app. Turn Wi-Fi back on. Verify the offline queue flushes to the Go backend and logs `[SYNC_COMPLETE]`.

3. **The Financial Anomaly Test:** 
   Open `http://localhost:3000/admin/ledger`. The Go `audit_cron` will fire at the top of the hour. *(To force immediately: temporarily change `admin/audit_cron.go` from `"0 * * * *"` to `"* * * * *"`).* Verify the pulsing red delta appears on the glass.

---
**Protocol Directives:** Execute the assault. If the matrix holds, Terraform deployment to Google Cloud production is authorized. Await post-test diagnostic report.