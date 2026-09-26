import { View, Text, ScrollView, Pressable, Switch } from 'react-native';
import { type ManifestItem } from '../utils/manifest';
import { type AppTheme, isIOS } from '../theme';
import { useState } from 'react';

// Extended to support LIFO sequences
export interface ManifestItemWithSequence extends ManifestItem {
  sequence_index?: number;
}

interface OrderChecklistProps {
  selectedManifest: ManifestItemWithSequence[];
  theme: AppTheme;
  toggleCheck: (itemId: string) => void;
}

export default function OrderChecklist({
  selectedManifest,
  theme: T,
  toggleCheck,
}: OrderChecklistProps) {
  const [overrideActive, setOverrideActive] = useState(false);

  // Sort by LIFO sequence index
  const sortedManifest = [...selectedManifest].sort(
    (a, b) => (a.sequence_index || 0) - (b.sequence_index || 0)
  );

  return (
    <View style={{ flex: 1 }}>
      {/* Supervisor Override Bar */}
      <View style={{ flexDirection: 'row', alignItems: 'center', justifyContent: 'flex-end', paddingHorizontal: 32, paddingBottom: 8 }}>
        <Text style={{ marginRight: 8, color: overrideActive ? T.colors.destructive : T.colors.secondaryLabel, fontWeight: 'bold' }}>
          SUPERVISOR OVERRIDE
        </Text>
        <Switch
          value={overrideActive}
          onValueChange={setOverrideActive}
          trackColor={{ true: T.colors.destructive }}
        />
      </View>

      <ScrollView className="flex-1 px-8 py-2">
        {sortedManifest.map((item, index) => {
          // LIFO Lock logic: must load in sequence unless overridden
          const isPreviousScanned = index === 0 || sortedManifest[index - 1].scanned;
          const isLocked = !isPreviousScanned && !overrideActive;

          return (
            <Pressable
              key={item.id}
              onPress={() => {
                if (!isLocked) toggleCheck(item.id);
              }}
              style={({ pressed }) => ({
                flexDirection: 'row' as const,
                alignItems: 'center' as const,
                paddingVertical: 16,
                borderBottomWidth: isIOS ? 0.33 : 1,
                borderBottomColor: T.colors.separator,
                opacity: isLocked ? 0.3 : item.scanned ? 0.4 : pressed ? 0.75 : 1,
                transform: [{ scale: pressed && !isLocked ? 0.99 : 1 }],
                backgroundColor: isLocked ? '#F3F4F6' : 'transparent',
              })}
            >
              {/* Checkbox */}
              <View style={{
                width: 22,
                height: 22,
                borderRadius: T.radius.checkbox,
                borderWidth: item.scanned ? 0 : (isIOS ? 1.5 : 2),
                borderColor: item.scanned ? 'transparent' : isLocked ? '#D1D5DB' : T.colors.tertiaryLabel,
                backgroundColor: item.scanned ? T.colors.accent : 'transparent',
                marginRight: 16,
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                {item.scanned && (
                  <Text style={{ color: '#FFFFFF', fontWeight: '700', fontSize: 12 }}>✓</Text>
                )}
                {isLocked && !item.scanned && (
                  <Text style={{ color: '#9CA3AF', fontSize: 10 }}>🔒</Text>
                )}
              </View>
              <View>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, letterSpacing: 0.5 }}>
                  {item.brand} (Seq: {item.sequence_index ?? index})
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
          );
        })}
      </ScrollView>
    </View>
  );
}
