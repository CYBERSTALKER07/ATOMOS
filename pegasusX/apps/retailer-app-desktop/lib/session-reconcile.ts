import { apiFetch } from "./auth";

const RETAILER_RECONCILE_PATHS = [
  "/v1/retailer/active-fulfillment",
  "/v1/retailer/pending-payments",
  "/v1/retailer/tracking",
] as const;

/** Refetch server-authoritative retailer snapshots after transport reconnect. */
export async function reconcileRetailerSession(): Promise<void> {
  await Promise.all(
    RETAILER_RECONCILE_PATHS.map(async (path) => {
      try {
        await apiFetch(path);
      } catch {
        // Best-effort; individual screens refetch on reconnect events too.
      }
    }),
  );
}
