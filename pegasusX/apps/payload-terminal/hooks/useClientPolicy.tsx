import { useCallback, useEffect, useState } from 'react';
import { Text, View } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';

import { PayloadTerminalApi } from '../api';
import { registerPayloadPushTokens } from '../pushRegistration';

// ─── Client policy (force-update / outdated banner) ───────────────────────────

export function useClientPolicy({ token }: { token: string | null }) {
  const [clientPolicyMessage, setClientPolicyMessage] = useState<string | null>(null);

  const fetchClientPolicy = useCallback(async () => {
    try {
      const policy = await PayloadTerminalApi.getClientPolicy('expo', '1.0.0');
      if (policy.outdated || policy.force_update) {
        let message = policy.force_update ? 'Update required' : 'Update available';
        if (policy.minimum_version) {
          message += ` — minimum version ${policy.minimum_version}`;
        }
        if (policy.defer_reason) {
          message += `. ${policy.defer_reason}`;
        }
        setClientPolicyMessage(message);
      } else {
        setClientPolicyMessage(null);
      }
    } catch {
      // Policy fetch is optional on local/dev stacks.
    }
  }, []);

  const renderClientPolicyBanner = () => {
    if (!clientPolicyMessage) return null;
    return (
      <View style={{
        backgroundColor: 'rgba(245, 158, 11, 0.14)',
        borderBottomWidth: 1,
        borderBottomColor: 'rgba(245, 158, 11, 0.4)',
        paddingHorizontal: 16,
        paddingVertical: 10,
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
      }}>
        <MaterialIcons name="warning-amber" size={18} color="#B45309" />
        <Text style={{ flex: 1, color: '#92400E', fontSize: 13, fontWeight: '600' }}>{clientPolicyMessage}</Text>
      </View>
    );
  };

  useEffect(() => {
    fetchClientPolicy();
  }, [fetchClientPolicy]);

  useEffect(() => {
    if (!token) return;
    fetchClientPolicy();
    void registerPayloadPushTokens();
  }, [token, fetchClientPolicy]);

  return { renderClientPolicyBanner };
}
