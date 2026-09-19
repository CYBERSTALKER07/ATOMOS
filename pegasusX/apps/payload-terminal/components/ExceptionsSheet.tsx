import { Modal, View, Text, Pressable, FlatList, TextInput, Alert } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { isIOS } from '../theme';
import PayloadStatePanel from './PayloadStatePanel';
import StatusBadge, { exceptionReasonTone } from './StatusBadge';
import { SkeletonList } from './SkeletonPulse';
import { useState } from 'react';
import { authFetch } from '../authSession';
import { API_BASE } from '../api';

export type ManifestExceptionItem = {
  exception_id: string;
  manifest_id: string;
  order_id: string;
  reason: string;
  attempt_count: number;
  escalated: boolean;
  created_at: string;
};

type ExceptionsSheetProps = {
  visible: boolean;
  onClose: () => void;
  onRefresh: () => void;
  loading: boolean;
  exceptions: ManifestExceptionItem[];
  theme: any; 
  activeManifestId?: string;
};

export default function ExceptionsSheet({ visible, onClose, onRefresh, loading, exceptions, theme: T, activeManifestId }: ExceptionsSheetProps) {
  const [damagedItemId, setDamagedItemId] = useState('');
  const [damagedOrderId, setDamagedOrderId] = useState('');

  const reportDamage = async () => {
    if (!activeManifestId || !damagedItemId || !damagedOrderId) {
      Alert.alert('Error', 'Missing required fields.');
      return;
    }
    try {
      const res = await authFetch(`${API_BASE}/v1/payload/exceptions/damaged`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          manifest_id: activeManifestId,
          order_id: damagedOrderId,
          item_id: damagedItemId,
          price_deduction: 50 // Example default deduction
        })
      });
      if (res.ok) {
        Alert.alert('Success', 'Item marked as damaged. Ledger and order updated globally.');
        setDamagedItemId('');
        setDamagedOrderId('');
        onRefresh();
      } else {
        Alert.alert('Error', 'Failed to report damage.');
      }
    } catch (e) {
      Alert.alert('Error', 'Network error.');
    }
  };

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.4)', justifyContent: 'center', alignItems: 'center' }}>
        <View style={{ width: 420, maxHeight: '80%', backgroundColor: T.colors.sidebarBackground, borderRadius: 12, overflow: 'hidden' }}>
          <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
            <Text style={{ flex: 1, fontWeight: '700', fontSize: 15, color: T.colors.sidebarLabel, letterSpacing: 0.3 }}>
              {isIOS ? 'Manifest exceptions' : 'MANIFEST EXCEPTIONS'}
            </Text>
            <Pressable onPress={onRefresh} style={{ marginRight: 12 }}>
              <MaterialIcons name="refresh" size={20} color={T.colors.accent} />
            </Pressable>
            <Pressable onPress={onClose}>
              <MaterialIcons name="close" size={20} color={T.colors.sidebarSecondary} />
            </Pressable>
          </View>
          
          {/* Report Damage Section */}
          <View style={{ padding: 16, backgroundColor: '#FEE2E2', borderBottomWidth: 1, borderBottomColor: '#FCA5A5' }}>
            <Text style={{ color: '#991B1B', fontWeight: 'bold', marginBottom: 8 }}>Report Dock Damage</Text>
            <TextInput
              placeholder="Order ID"
              value={damagedOrderId}
              onChangeText={setDamagedOrderId}
              style={{ backgroundColor: 'white', padding: 8, borderRadius: 4, marginBottom: 8 }}
            />
            <TextInput
              placeholder="Item ID"
              value={damagedItemId}
              onChangeText={setDamagedItemId}
              style={{ backgroundColor: 'white', padding: 8, borderRadius: 4, marginBottom: 8 }}
            />
            <Pressable 
              onPress={reportDamage}
              style={{ backgroundColor: '#DC2626', padding: 10, borderRadius: 6, alignItems: 'center' }}
            >
              <Text style={{ color: 'white', fontWeight: 'bold' }}>MARK DAMAGED</Text>
            </Pressable>
          </View>

          {loading && exceptions.length === 0 ? (
            <SkeletonList count={4} theme={T} />
          ) : (
            <FlatList
              data={exceptions}
              keyExtractor={item => item.exception_id}
              ListEmptyComponent={
                <View style={{ padding: 40, alignItems: 'center' }}>
                  <PayloadStatePanel
                    theme={T}
                    variant="manifest"
                    title={isIOS ? 'No exceptions' : 'NO EXCEPTIONS'}
                    message={isIOS ? 'Overflow, damaged, and manual removals appear here.' : 'OVERFLOW, DAMAGED, AND MANUAL REMOVALS APPEAR HERE.'}
                    compact
                  />
                </View>
              }
              renderItem={({ item }) => (
                <View style={{ paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
                  <View style={{ flexDirection: 'row', alignItems: 'center', marginBottom: 4, gap: 8 }}>
                    <StatusBadge compact label={item.reason} theme={T} tone={exceptionReasonTone(item.reason)} />
                    {item.escalated ? (
                      <StatusBadge compact label={isIOS ? 'Escalated' : 'ESCALATED'} theme={T} tone="danger" />
                    ) : null}
                  </View>
                  <Text style={{ fontSize: 12, color: T.colors.sidebarSecondary }}>
                    {isIOS ? 'Order' : 'ORDER'} {item.order_id.slice(0, 8)} · {isIOS ? 'Manifest' : 'MANIFEST'} {item.manifest_id.slice(0, 8)}
                  </Text>
                  <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4 }}>
                    {isIOS ? 'Attempts' : 'ATTEMPTS'} {item.attempt_count}
                  </Text>
                </View>
              )}
            />
          )}
        </View>
      </View>
    </Modal>
  );
}
