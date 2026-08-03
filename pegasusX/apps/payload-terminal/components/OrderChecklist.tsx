import { View, Text, ScrollView, Pressable } from 'react-native';
import { type ManifestItem } from '../utils/manifest';
import { type AppTheme, isIOS } from '../theme';

interface OrderChecklistProps {
  selectedManifest: ManifestItem[];
  theme: AppTheme;
  toggleCheck: (itemId: string) => void;
}

export default function OrderChecklist({
  selectedManifest,
  theme: T,
  toggleCheck,
}: OrderChecklistProps) {
  return (
    <ScrollView className="flex-1 px-8 py-2">
      {selectedManifest.map(item => (
        <Pressable
          key={item.id}
          onPress={() => toggleCheck(item.id)}
          style={({ pressed }) => ({
            flexDirection: 'row' as const,
            alignItems: 'center' as const,
            paddingVertical: 16,
            borderBottomWidth: isIOS ? 0.33 : 1,
            borderBottomColor: T.colors.separator,
            opacity: item.scanned ? 0.4 : pressed ? 0.75 : 1,
            transform: [{ scale: pressed ? 0.99 : 1 }],
          })}
        >
          {/* Checkbox */}
          <View style={{
            width: 22,
            height: 22,
            borderRadius: T.radius.checkbox,
            borderWidth: item.scanned ? 0 : (isIOS ? 1.5 : 2),
            borderColor: item.scanned ? 'transparent' : T.colors.tertiaryLabel,
            backgroundColor: item.scanned ? T.colors.accent : 'transparent',
            marginRight: 16,
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            {item.scanned && (
              <Text style={{ color: '#FFFFFF', fontWeight: '700', fontSize: 12 }}>✓</Text>
            )}
          </View>
          <View>
            <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, letterSpacing: 0.5 }}>
              {item.brand}
            </Text>
            <View style={{ flexDirection: 'row', alignItems: 'center', marginTop: 2 }}>
              <Text style={{ fontWeight: '600', fontSize: 15, color: T.colors.label }}>
                {item.label}
              </Text>
              <Text style={{ fontWeight: '500', fontSize: 13, color: item.scanned ? '#16A34A' : T.colors.secondaryLabel, marginLeft: 8 }}>
                ({item.verifiedQuantity}/{item.quantity} scanned)
              </Text>
            </View>
          </View>
        </Pressable>
      ))}
    </ScrollView>
  );
}
