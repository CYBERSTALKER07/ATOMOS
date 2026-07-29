import { Modal, View, Text, FlatList, Pressable } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { isIOS } from '../theme';
import PayloadStatePanel from './PayloadStatePanel';

export type NotifItem = {
  id: string;
  type: string;
  title: string;
  body: string;
  read_at: string | null;
  created_at: string;
};

export interface NotificationsSheetProps {
  visible: boolean;
  onClose: () => void;
  theme: any;
  notifications: NotifItem[];
  unreadCount: number;
  onMarkAllRead: () => void;
  onMarkRead: (id: string) => void;
}

export default function NotificationsSheet({
  visible,
  onClose,
  theme: T,
  notifications,
  unreadCount,
  onMarkAllRead,
  onMarkRead,
}: NotificationsSheetProps) {
  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.4)', justifyContent: 'center', alignItems: 'center' }}>
        <View style={{ width: 420, maxHeight: '80%', backgroundColor: T.colors.sidebarBackground, borderRadius: 12, overflow: 'hidden' }}>
          {/* Modal header */}
          <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
            <Text style={{ flex: 1, fontWeight: '700', fontSize: 15, color: T.colors.sidebarLabel, letterSpacing: 0.3 }}>
              {isIOS ? 'Notifications' : 'NOTIFICATIONS'}
            </Text>
            {unreadCount > 0 && (
              <Pressable onPress={onMarkAllRead} style={{ marginRight: 12 }}>
                <Text style={{ fontSize: 12, color: T.colors.accent, fontWeight: '600' }}>Mark all read</Text>
              </Pressable>
            )}
            <Pressable onPress={onClose}>
              <MaterialIcons name="close" size={20} color={T.colors.sidebarSecondary} />
            </Pressable>
          </View>
          {/* Notification list */}
          <FlatList
            data={notifications}
            keyExtractor={item => item.id}
            ListEmptyComponent={
              <View style={{ padding: 40, alignItems: 'center' }}>
                <PayloadStatePanel
                  theme={T}
                  variant="notifications"
                  title={isIOS ? 'No notifications' : 'NO NOTIFICATIONS'}
                  message={isIOS ? 'Payload alerts and sync events will appear here.' : 'PAYLOAD ALERTS AND SYNC EVENTS WILL APPEAR HERE.'}
                  compact
                />
              </View>
            }
            renderItem={({ item }) => {
              const isUnread = !item.read_at;
              const iconName: keyof typeof MaterialIcons.glyphMap =
                item.type === 'PAYLOAD_READY_TO_SEAL' ? 'inventory' :
                item.type === 'PAYLOAD_SEALED' ? 'verified' :
                item.type === 'ORDER_DISPATCHED' ? 'local-shipping' :
                item.type === 'ORDER_COMPLETED' ? 'check-circle' :
                item.type === 'PAYMENT_SETTLED' ? 'payments' :
                item.type === 'PAYMENT_FAILED' ? 'error' :
                'notifications';
              return (
                <Pressable
                  onPress={() => { if (isUnread) onMarkRead(item.id); }}
                  style={{
                    flexDirection: 'row',
                    paddingHorizontal: 20,
                    paddingVertical: 12,
                    borderBottomWidth: 0.5,
                    borderBottomColor: T.colors.sidebarSeparator,
                    backgroundColor: isUnread ? `${T.colors.accent}10` : 'transparent',
                  }}
                >
                  <MaterialIcons name={iconName} size={18} color={isUnread ? T.colors.accent : T.colors.sidebarSecondary} style={{ marginRight: 12, marginTop: 2 }} />
                  <View style={{ flex: 1 }}>
                    <Text style={{ fontWeight: isUnread ? '700' : '500', fontSize: 13, color: T.colors.sidebarLabel, marginBottom: 2 }}>{item.title}</Text>
                    <Text style={{ fontSize: 12, color: T.colors.sidebarSecondary }} numberOfLines={2}>{item.body}</Text>
                    <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4 }}>{new Date(item.created_at).toLocaleString()}</Text>
                  </View>
                  {isUnread && <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: T.colors.accent, alignSelf: 'center', marginLeft: 8 }} />}
                </Pressable>
              );
            }}
          />
        </View>
      </View>
    </Modal>
  );
}
