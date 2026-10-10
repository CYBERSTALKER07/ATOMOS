import type { ReactNode } from 'react';
import { Text, View } from 'react-native';

import { isIOS, type AppTheme } from '../theme';
import Pressable from './Pressable';
import TruckSidebar, { type Truck } from './TruckSidebar';

// ─── Render: AWAITING TRUCK SELECTION ─────────────────────────────────────────

export default function TruckSelectionScreen({
  theme: T,
  tx,
  workerName,
  trucks,
  activeTruck,
  isLoadingTrucks,
  onTruckSelect,
  onLogout,
  onShowInbound,
  policyBanner,
  toast,
}: {
  theme: AppTheme;
  tx: (key: string) => string;
  workerName: string;
  trucks: Truck[];
  activeTruck: string | null;
  isLoadingTrucks: boolean;
  onTruckSelect: (truckId: string) => void;
  onLogout: () => void;
  onShowInbound: () => void;
  policyBanner: ReactNode;
  toast: ReactNode;
}) {
  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background }}>
      {policyBanner}
      {/* Header */}
      <View style={{ backgroundColor: T.colors.sidebarBackground, paddingHorizontal: 32, paddingVertical: 16, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
        <View>
          <Text style={{ color: T.colors.sidebarLabel, fontWeight: '700', fontSize: 14, letterSpacing: 0.3 }}>
            {tx('auth.login.payload_terminal')}
          </Text>
          <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 11, marginTop: 2 }}>
            {workerName}
          </Text>
        </View>
        <View style={{ flexDirection: 'row', alignItems: 'center' }}>
          <Pressable onPress={onShowInbound} style={{ paddingHorizontal: 16, paddingVertical: 8, marginRight: 8 }}>
            <Text style={{ color: T.colors.sidebarLabel, fontSize: 12, fontWeight: '700', letterSpacing: 0.3 }}>
              {isIOS ? 'Inbound Returns' : 'INBOUND RETURNS'}
            </Text>
          </Pressable>
          <Pressable onPress={onLogout} style={{ paddingHorizontal: 16, paddingVertical: 8 }}>
            <Text style={{ color: T.colors.sidebarSecondary, fontSize: 12, fontWeight: '600', letterSpacing: 0.3 }}>
              {tx('common.action.sign_out')}
            </Text>
          </Pressable>
        </View>
      </View>

      {/* Truck selector */}
      <TruckSidebar
        variant="selector"
        trucks={trucks}
        activeTruck={activeTruck}
        setActiveTruck={onTruckSelect}
        isLoadingTrucks={isLoadingTrucks}
        tx={tx}
      />
      {toast}
    </View>
  );
}
