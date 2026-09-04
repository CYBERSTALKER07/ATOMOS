import type { ReactNode } from 'react';
import { Text, View } from 'react-native';

import { isIOS, type AppTheme } from '../theme';
import Pressable from './Pressable';

// ─── Render: ALL SEALED ───────────────────────────────────────────────────────

export default function AllSealedScreen({
  theme: T,
  activeTruck,
  dispatchCodes,
  onNewManifest,
  toast,
}: {
  theme: AppTheme;
  activeTruck: string | null;
  dispatchCodes: Record<string, string>;
  onNewManifest: () => void;
  toast: ReactNode;
}) {
  return (
    <View style={{ flex: 1, backgroundColor: T.colors.success, alignItems: 'center', justifyContent: 'center', padding: 48 }}>
      <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: 'rgba(255,255,255,0.7)', letterSpacing: 0.5, marginBottom: 24 }}>
        {activeTruck}
      </Text>
      <Text style={{ fontSize: 32, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.6 : 0.5, marginBottom: 8 }}>
        {isIOS ? 'Manifest Secured.' : 'MANIFEST SECURED.'}
      </Text>
      <Text style={{ fontSize: 32, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.6 : 0.5, marginBottom: 24 }}>
        {isIOS ? 'Fleet Dispatched.' : 'FLEET DISPATCHED.'}
      </Text>

      {/* Dispatch codes (JIT QR substitute) */}
      {Object.keys(dispatchCodes).length > 0 && (
        <View style={{ backgroundColor: 'rgba(255,255,255,0.15)', borderRadius: 16, padding: 20, marginBottom: 32, minWidth: 280 }}>
          <Text style={{ fontSize: 11, fontWeight: '700', color: 'rgba(255,255,255,0.7)', letterSpacing: 1, textAlign: 'center', marginBottom: 12 }}>
            {isIOS ? 'Dispatch Codes' : 'DISPATCH CODES'}
          </Text>
          {Object.entries(dispatchCodes).map(([orderId, code]) => (
            <View key={orderId} style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 8, borderBottomWidth: 0.5, borderBottomColor: 'rgba(255,255,255,0.2)' }}>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 12, color: 'rgba(255,255,255,0.8)' }}>{orderId}</Text>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 16, fontWeight: '700', color: '#FFFFFF', letterSpacing: 2 }}>{code}</Text>
            </View>
          ))}
        </View>
      )}
      <Pressable
        onPress={onNewManifest}
        style={({ pressed }) => ({
          borderWidth: 1,
          borderColor: 'rgba(255,255,255,0.5)',
          paddingHorizontal: 32,
          paddingVertical: 14,
          borderRadius: T.radius.button,
          opacity: pressed ? 0.82 : 1,
          transform: [{ scale: pressed ? 0.97 : 1 }],
        })}
      >
        <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
          {isIOS ? 'New Manifest' : 'NEW MANIFEST'}
        </Text>
      </Pressable>
      {toast}
    </View>
  );
}
