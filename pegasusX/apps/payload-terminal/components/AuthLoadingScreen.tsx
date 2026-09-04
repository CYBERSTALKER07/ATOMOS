import type { ReactNode } from 'react';
import { View } from 'react-native';

import PayloadStatePanel from './PayloadStatePanel';
import { isIOS, type AppTheme } from '../theme';

// ─── Render: AUTH LOADING ─────────────────────────────────────────────────────

export default function AuthLoadingScreen({
  theme: T,
  tx,
  toast,
}: {
  theme: AppTheme;
  tx: (key: string) => string;
  toast: ReactNode;
}) {
  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background, alignItems: 'center', justifyContent: 'center' }}>
      <PayloadStatePanel
        theme={T}
        variant="sync"
        title={tx('common.status.restoring_session')}
        message={isIOS ? 'Rehydrating the saved operator session and pending queue.' : 'REHYDRATING THE SAVED OPERATOR SESSION AND PENDING QUEUE.'}
      />
      {toast}
    </View>
  );
}
