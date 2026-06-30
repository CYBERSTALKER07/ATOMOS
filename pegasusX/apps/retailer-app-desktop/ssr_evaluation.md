# Server-Side Rendering (SSR) Evaluation for Retailer Desktop

## Context
The Retailer Desktop application (`apps/retailer-app-desktop`) is built using Next.js (App Router) wrapped in a Tauri shell to provide a native desktop experience. The architecture requires `output: "export"` in `next.config.ts` to bundle a static HTML/JS payload that Tauri can render in its webview without a Node.js server.

## Problem Statement
We need to evaluate the feasibility of enabling Server-Side Rendering (SSR) for the Retailer Desktop dashboard to improve initial load performance, SEO (if applicable), and reduce client-side data fetching spinners.

## Analysis

### Technical Constraints
1. **Tauri Requires Static Assets**: Tauri serves the application from the local filesystem (`file://` or custom local protocol). It does not embed a Node.js runtime to execute server-side React code at request time.
2. **Next.js Static Export Limitations**: Using `output: "export"` disables all SSR features (like `getServerSideProps` in Pages router, or dynamic Server Components fetching request-time data in App router). Any data fetching in Server Components is executed at *build time* (SSG).

### The SSR "Fallback"
If we host the Next.js app on a standard web server (e.g., Vercel or a Node server) and point the Tauri webview to that remote URL instead of bundling the app locally:
- **Pros**: Full SSR capabilities, instant updates without distributing new binaries.
- **Cons**: Requires constant internet connection (breaks the offline-first requirement), increases latency depending on server proximity, and loses the security/isolation benefits of a locally bundled Tauri app.

## Alternatives to SSR

Given that true SSR is incompatible with the bundled Tauri architecture, we recommend the following strategies to achieve similar UX goals:

### 1. Static Site Generation (SSG) with Client-Side Hydration
Use React Server Components to fetch static data at build time (or rely on static shells). The UI initially renders immediately (like SSR), and client components hydrate dynamic data (e.g., orders, live stock) on mount using SWR or React Query (we currently use `useLiveData`).

### 2. Local SQLite Caching (Recommended)
Integrate a local SQLite database using Tauri's SQL plugin. 
- The app instantly loads data from the local database on startup (zero network latency, simulating SSR-like speed).
- A background worker synchronizes with the remote server.

### 3. Optimistic UI and Skeleton Screens
Enhance the existing `PageChrome` and `BentoGrid` components with robust skeleton loaders. While data fetches on mount, the user sees a structurally complete UI rather than a blank screen.

## Conclusion
True SSR is not recommended and technically infeasible for the current Tauri-based Retailer Desktop app if we want to maintain it as a standalone, offline-capable binary. We should lean into local caching (SQLite) and static shell rendering to achieve the instant-load feel of SSR without compromising the desktop architecture.
