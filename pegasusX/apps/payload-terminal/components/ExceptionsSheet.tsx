import { Modal, View, Text, Pressable, FlatList } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { isIOS } from '../theme';
import PayloadStatePanel from './PayloadStatePanel';
import StatusBadge, { exceptionReasonTone } from './StatusBadge';
import { SkeletonList } from './SkeletonPulse';

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
  theme: any; // Using any or importing Theme type if available. For now using any to match App.tsx structure where it passes T
};

export default function ExceptionsSheet({ visible, onClose, onRefresh, loading, exceptions, theme: T }: ExceptionsSheetProps) {
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
