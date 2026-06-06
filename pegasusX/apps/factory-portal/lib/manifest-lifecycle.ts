export type ManifestLifecycleAction = {
  path: 'start-loading' | 'seal' | 'dispatch' | 'complete';
  label: string;
};

export function nextManifestLifecycleAction(state: string): ManifestLifecycleAction | null {
  switch (state.toUpperCase()) {
    case 'DRAFT':
      return { path: 'start-loading', label: 'Start loading' };
    case 'LOADING':
      return { path: 'seal', label: 'Seal manifest' };
    case 'SEALED':
      return { path: 'dispatch', label: 'Dispatch' };
    case 'DISPATCHED':
      return { path: 'complete', label: 'Complete' };
    default:
      return null;
  }
}

export const MANIFEST_STATE_ORDER = ['DRAFT', 'LOADING', 'SEALED', 'DISPATCHED', 'COMPLETED', 'CANCELLED'] as const;
