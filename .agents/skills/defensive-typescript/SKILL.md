# Defensive TypeScript — Frontend Anti-Pattern Prevention

## Description
Prevents silent failures, type safety erosion, configuration drift, and UX incompleteness in TypeScript/React/Next.js web applications. Activates when writing or reviewing code in the admin portal, retailer desktop, factory portal, or warehouse portal.

## Trigger Keywords
typescript, react, next.js, nextjs, component, page, hook, useEffect, useState, fetch, API, portal, admin-portal, retailer-desktop, factory-portal, warehouse-portal, tailwind, error boundary

## Anti-Pattern Catalog

### 1. API Base URL Duplication (CONFIGURATION DRIFT)
```typescript
// ❌ WRONG — copy-pasted in 20+ files
const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const res = await fetch(`${API}/v1/orders`);

// ✅ RIGHT — single source of truth
// lib/config.ts
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
export const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080';
export const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';

// In components:
import { API_BASE } from '@/lib/config';
const res = await fetch(`${API_BASE}/v1/orders`);
```
**Real finding**: `process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'` duplicated in `lib/auth.ts`, `hooks/useTelemetry.ts`, `app/page.tsx`, `app/fleet/page.tsx`, `app/treasury/*/page.tsx`, `app/auth/*/page.tsx`, and 15+ more files. CartoDB map style URL duplicated across 4 files.

### 2. Silent Error Swallowing (.catch(() => {}))
```typescript
// ❌ WRONG — error silently swallowed, impossible to diagnose
await updateShift(isActive).catch(() => {});

// ✅ RIGHT — log at minimum
await updateShift(isActive).catch((err) => {
    console.error('shift toggle failed:', err);
});

// ✅ BETTER — surface to user for user-facing operations
try {
    await updateShift(isActive);
} catch (err) {
    setError('Failed to update shift status. Please retry.');
    console.error('shift toggle failed:', err);
}
```
**Real findings**: 9 `.catch(() => {})` instances — `lib/auth.ts` (token storage), `lib/usePolling.ts`, `lib/useLiveData.ts`, `hooks/useSupplierShift.tsx`, `app/fleet/page.tsx` (telemetry disconnect), `app/supplier/orders/page.tsx` (order actions).

**Rule**: Never `.catch(() => {})`. Acceptable patterns:
- Background cleanup: `.catch((err) => console.error('cleanup:', err))`
- User-facing mutation: `try/catch` with UI error state
- AbortController cleanup: `.catch(() => {})` is OK only if the abort is the expected path

### 3. Missing Route-Level Error Boundaries
```typescript
// ❌ WRONG — only root error.tsx exists
// app/error.tsx ← catches ALL errors, nukes entire app

// ✅ RIGHT — nested error boundaries per route group
// app/fleet/error.tsx ← catches fleet errors, preserves app shell
// app/treasury/error.tsx
// app/supplier/error.tsx
// app/not-found.tsx ← 404 handling

// app/fleet/error.tsx
'use client';
export default function FleetError({ error, reset }: { error: Error; reset: () => void }) {
    return (
        <div className="md-card p-6">
            <h2 className="md-typescale-title-medium">Fleet data unavailable</h2>
            <p className="md-typescale-body-medium" style={{ color: 'var(--color-md-error)' }}>
                {error.message}
            </p>
            <button className="md-btn md-btn-filled mt-4" onClick={reset}>
                Retry
            </button>
        </div>
    );
}
```
**Real finding**: Only `app/error.tsx` exists. No nested `error.tsx` in `/fleet`, `/treasury`, `/supplier/*`. No `not-found.tsx`.

### 4. Tauri IPC: JS invoke() With No Rust Handler (SILENT FAILURE)
```typescript
// ❌ WRONG — invoke() calls silently fail if no Rust handler exists
import { invoke } from '@tauri-apps/api/core';
await invoke('store_token', { token }); // no Rust handler → silently fails
// tokens are NOT persisted, users lose session on restart

// ✅ RIGHT — every invoke() has a matching Rust handler
// src-tauri/src/lib.rs
#[tauri::command]
fn store_token(token: String) -> Result<(), String> { /* ... */ }

tauri::Builder::default()
    .invoke_handler(tauri::generate_handler![store_token, get_stored_token, clear_stored_token])
    .run(tauri::generate_context!())
```
**Real finding**: `retailer-app-desktop/lib/bridge.ts` calls `invoke('store_token')`, `invoke('get_stored_token')`, `invoke('clear_stored_token')`. `src-tauri/src/lib.rs` has zero registered command handlers. Auth tokens are silently not persisted.

### 5. `any` at Domain Boundaries (TYPE SAFETY LOSS)
```typescript
// ❌ WRONG — domain data loses type safety
const [orders, setOrders] = useState<any[]>([]);
const handleAction = (order: any) => { ... };

// ✅ RIGHT — typed from packages/types
import type { Order } from '@repo/types';
const [orders, setOrders] = useState<Order[]>([]);
const handleAction = (order: Order) => { ... };
```
**Exception**: `as any` for third-party library interop is acceptable (Recharts callbacks, WebGL libraries, Firebase internal APIs). Never `any` for: API responses, store state, component props, or event handlers.

### 6. Five-State Data Surface (NO FAKE COMPLETENESS)
```typescript
// ❌ WRONG — only shows data or blank
function OrdersPage() {
    const { data } = useSWR('/api/orders');
    return <OrderTable orders={data} />;
}

// ✅ RIGHT — all five states
function OrdersPage() {
    const { data, error, isLoading, isValidating } = useSWR('/api/orders');

    if (isLoading) return <OrderTableSkeleton />;
    if (error) return <ErrorCard message="Failed to load orders" onRetry={mutate} />;
    if (!data || data.length === 0) return <EmptyState icon="inbox" message="No orders yet" />;

    return (
        <>
            {isValidating && <StaleDataBanner />}
            <OrderTable orders={data} />
        </>
    );
}
```
**Five states**: (1) loading, (2) empty, (3) error, (4) stale/refreshing, (5) success with data.

### 7. useEffect Dependency Hygiene
```typescript
// ❌ WRONG — missing dependency causes stale closure
useEffect(() => {
    const interval = setInterval(() => {
        fetchOrders(supplierId); // supplierId is stale if it changes
    }, 5000);
    return () => clearInterval(interval);
}, []); // supplierId not in deps

// ✅ RIGHT — include all dependencies
const fetchOrdersCallback = useCallback(() => {
    fetchOrders(supplierId);
}, [supplierId]);

useEffect(() => {
    const interval = setInterval(fetchOrdersCallback, 5000);
    return () => clearInterval(interval);
}, [fetchOrdersCallback]);
```

### 8. XSS Prevention
```typescript
// ❌ WRONG — user input rendered as HTML
<div dangerouslySetInnerHTML={{ __html: userComment }} />

// ✅ RIGHT — render as text (React auto-escapes)
<div>{userComment}</div>

// If HTML is truly needed (e.g., rich text editor output):
import DOMPurify from 'dompurify';
<div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(richTextContent) }} />
```
**Rule**: `dangerouslySetInnerHTML` with user-provided content is an XSS vector. React's default JSX rendering escapes HTML. Only use `dangerouslySetInnerHTML` for trusted, sanitized content.

## Type Alignment with Backend
```typescript
// packages/types/entities.ts MUST match backend Go structs
// Check for drift: if backend adds HomeNodeType/HomeNodeId, TS must follow

// ❌ STALE — missing fields backend writes
interface Driver {
    driver_id: string;
    name: string;
    supplier_id: string;
    // HomeNodeType? HomeNodeId? — MISSING
}

// ✅ ALIGNED
interface Driver {
    driver_id: string;
    name: string;
    supplier_id: string;
    home_node_type: 'WAREHOUSE' | 'FACTORY';
    home_node_id: string;
    warehouse_id: string; // legacy compat
}
```

## Verification Checklist
- [ ] API base URL comes from `lib/config.ts`, not inline `process.env`
- [ ] Zero `.catch(() => {})` without logging or explicit justification
- [ ] Route-level `error.tsx` exists for every major route group
- [ ] Every Tauri `invoke()` has a matching Rust `#[tauri::command]` handler
- [ ] No `any` at domain boundaries (API responses, props, state)
- [ ] Every data-fetching component handles all five states
- [ ] No `dangerouslySetInnerHTML` with unsanitized user input
- [ ] Types in `packages/types` match current backend Go structs

## Cross-References
- **intrusions.md** §9 (Frontend Traps) — full finding details
- **gemini-instructions.md** Web UI Stack — M3 theming, component patterns
- **gemini-instructions.md** Surface Completeness — five-state requirement
- **design-system** skill — Leviathan Retailer Desktop design tokens
