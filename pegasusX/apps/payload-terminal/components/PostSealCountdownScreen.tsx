import type { ReactNode } from 'react';
import { Alert, Text, View } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';

import { authFetch } from '../authSession';
import { isIOS, type AppTheme } from '../theme';
import { payloadMissingItemsKey } from '../utils/idempotency';
import Pressable from './Pressable';
import type { ShowToast } from '../hooks/useToast';

// ─── Render: POST-SEAL DOUBLE-CHECK COUNTDOWN (Edge 33) ───────────────────────

export default function PostSealCountdownScreen({
  theme: T,
  postSealOrderId,
  postSealCountdown,
  showToast,
  toast,
}: {
  theme: AppTheme;
  postSealOrderId: string | null;
  postSealCountdown: number;
  showToast: ShowToast;
  toast: ReactNode;
}) {
  return (
    <View style={{ flex: 1, backgroundColor: (T.colors as any)?.warning ?? '#F59E0B', alignItems: 'center', justifyContent: 'center', padding: 48 }}>
      <MaterialIcons name="verified-user" size={64} color="rgba(255,255,255,0.9)" style={{ marginBottom: 24 }} />
      <Text style={{ fontSize: 28, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.4 : 0.5, marginBottom: 8 }}>
        {isIOS ? 'Double-Check' : 'DOUBLE-CHECK'}
      </Text>
      <Text style={{ fontSize: 14, color: 'rgba(255,255,255,0.85)', textAlign: 'center', maxWidth: 400, marginBottom: 32 }}>
        {isIOS
          ? 'Verify the sealed order before it dispatches. Report missing items now if anything was forgotten.'
          : 'VERIFY THE SEALED ORDER BEFORE IT DISPATCHES. REPORT MISSING ITEMS NOW IF ANYTHING WAS FORGOTTEN.'}
      </Text>
      <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 72, fontWeight: '800', color: '#FFFFFF', marginBottom: 32 }}>
        {postSealCountdown}
      </Text>
      <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 12, color: 'rgba(255,255,255,0.7)', letterSpacing: 0.5, marginBottom: 32 }}>
        {postSealOrderId}
      </Text>
      <Pressable
        onPress={() => {
          Alert.alert(
            isIOS ? 'Report Missing Items?' : 'REPORT MISSING ITEMS?',
            isIOS
              ? 'This will flag the sealed order for review. The driver will be notified to verify the load.'
              : 'THIS WILL FLAG THE SEALED ORDER FOR REVIEW. THE DRIVER WILL BE NOTIFIED TO VERIFY THE LOAD.',
            [
              { text: isIOS ? 'Cancel' : 'CANCEL', style: 'cancel' },
              {
                text: isIOS ? 'Report' : 'REPORT',
                style: 'destructive',
                onPress: async () => {
                  try {
                    await authFetch('/v1/delivery/missing-items', {
                      method: 'POST',
                      headers: {
                        'Idempotency-Key': payloadMissingItemsKey(postSealOrderId ?? 'unknown-order'),
                      },
                      body: JSON.stringify({ order_id: postSealOrderId, items: [], source: 'PAYLOAD_TERMINAL' }),
                    });
                    await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
                    showToast(
                      isIOS ? 'Reported' : 'REPORTED',
                      isIOS ? 'Missing items flagged for review.' : 'MISSING ITEMS FLAGGED FOR REVIEW.',
                      'warning'
                    );
                  } catch (e: unknown) {
                    showToast('ERROR', e instanceof Error ? e.message : 'Failed to report', 'error');
                  }
                },
              },
            ]
          );
        }}
        style={({ pressed }) => ({
          borderWidth: 2,
          borderColor: 'rgba(255,255,255,0.6)',
          paddingHorizontal: 32,
          paddingVertical: 14,
          borderRadius: T.radius.button,
          flexDirection: 'row' as const,
          alignItems: 'center' as const,
          gap: 8,
          opacity: pressed ? 0.82 : 1,
          transform: [{ scale: pressed ? 0.97 : 1 }],
        })}
      >
        <MaterialIcons name="report-problem" size={20} color="#FFFFFF" />
        <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
          {isIOS ? 'Report Missing Items' : 'REPORT MISSING ITEMS'}
        </Text>
      </Pressable>
      {toast}
    </View>
  );
}
