import {
  factoryManifestCompleteKey,
  factoryManifestDispatchKey,
  factoryManifestSealKey,
  factoryManifestStartLoadingKey,
} from '@pegasusx/api-client';

export type ManifestLifecycleAction = {
  path: 'start-loading' | 'seal' | 'dispatch' | 'complete';
  label: string;
};

export function manifestTransitionIdempotencyKey(
  manifestId: string,
  action: ManifestLifecycleAction['path'],
  factoryId = '',
): string {
  switch (action) {
    case 'start-loading':
      return factoryManifestStartLoadingKey(manifestId);
    case 'seal':
      return factoryManifestSealKey(manifestId, factoryId);
    case 'dispatch':
      return factoryManifestDispatchKey(manifestId, factoryId);
    case 'complete':
      return factoryManifestCompleteKey(manifestId, factoryId);
    default:
      return `factory-manifest-transition:${factoryId}:${manifestId}:${action}`;
  }
}

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
