import { authFetch } from './authSession';

const PAYLOAD_RECONCILE_PATHS = [
  '/v1/payloader/trucks',
  '/v1/payloader/manifests',
] as const;

/** Refetch server-authoritative payload snapshots after transport reconnect. */
export async function reconcilePayloadSession(): Promise<void> {
  await Promise.all(
    PAYLOAD_RECONCILE_PATHS.map(async (path) => {
      try {
        await authFetch(path);
      } catch {
        // Best-effort; UI refresh follows reconcile.
      }
    }),
  );
}
