import { View, Text, Pressable } from 'react-native';
import { isIOS } from '../theme';
import StatusBadge from './StatusBadge';

export type TruckRecommendation = {
  driver_id: string;
  driver_name: string;
  vehicle_id: string;
  vehicle_class: string;
  license_plate: string;
  max_volume_vu: number;
  used_volume_vu: number;
  free_volume_vu: number;
  distance_km: number;
  order_count: number;
  truck_status: string;
  score: number;
  recommendation: string;
};

type Props = {
  item: TruckRecommendation;
  index: number;
  reDispatchVolume: number;
  isReassigning: boolean;
  theme: any;
  onReassign: (driverId: string, vehicleId: string) => void;
};

export default function RecommendationCard({
  item,
  index,
  reDispatchVolume,
  isReassigning,
  theme: T,
  onReassign
}: Props) {
  const isBest = index === 0;
  const fits = item.free_volume_vu >= reDispatchVolume;
  const isMaintenance = item.truck_status === 'MAINTENANCE';

  return (
    <Pressable
      onPress={() => { if (!isMaintenance && !isReassigning) onReassign(item.driver_id, item.vehicle_id); }}
      disabled={isMaintenance || isReassigning}
      style={({ pressed }) => ({
        flexDirection: 'row',
        alignItems: 'center',
        paddingHorizontal: 24,
        paddingVertical: 16,
        borderBottomWidth: isIOS ? 0.33 : 1,
        borderBottomColor: T.colors.separator,
        opacity: pressed ? 0.82 : (isMaintenance ? 0.4 : 1),
        transform: pressed ? [{ scale: 0.97 }] : [],
        backgroundColor: isBest ? `${T.colors.accent}08` : 'transparent',
      })}
    >
      {/* Rank badge */}
      <View style={{
        width: 28,
        height: 28,
        borderRadius: 14,
        backgroundColor: isBest ? T.colors.accent : T.colors.fillTertiary,
        alignItems: 'center',
        justifyContent: 'center',
        marginRight: 16,
      }}>
        <Text style={{ fontWeight: '700', fontSize: 12, color: isBest ? '#FFFFFF' : T.colors.secondaryLabel }}>
          {index + 1}
        </Text>
      </View>

      {/* Truck info */}
      <View style={{ flex: 1 }}>
        <View style={{ flexDirection: 'row', alignItems: 'center' }}>
          <Text style={{ fontWeight: '600', fontSize: 14, color: T.colors.label }}>
            {item.driver_name}
          </Text>
          {isBest && (
            <StatusBadge compact label={isIOS ? 'Best' : 'BEST'} theme={T} tone="accent" />
          )}
          {isMaintenance && (
            <StatusBadge compact label={isIOS ? 'Maintenance' : 'MAINTENANCE'} theme={T} tone="danger" />
          )}
        </View>
        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, marginTop: 3, letterSpacing: 0.3 }}>
          {item.license_plate} · {item.vehicle_class}
        </Text>
        <Text style={{ fontSize: 11, color: T.colors.secondaryLabel, marginTop: 2 }}>
          {item.recommendation}
        </Text>
      </View>

      {/* Metrics */}
      <View style={{ alignItems: 'flex-end', marginLeft: 12 }}>
        {item.distance_km >= 0 ? (
          <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 13, fontWeight: '600', color: T.colors.label }}>
            {item.distance_km < 1 ? `${(item.distance_km * 1000).toFixed(0)}m` : `${item.distance_km.toFixed(1)}km`}
          </Text>
        ) : (
          <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel }}>
            {isIOS ? 'No GPS' : 'NO GPS'}
          </Text>
        )}
        <Text style={{
          fontFamily: T.typography.mono.fontFamily,
          fontSize: 11,
          marginTop: 2,
          color: fits ? T.colors.success : T.colors.destructive,
        }}>
          {item.free_volume_vu.toFixed(1)} VU free
        </Text>
        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 1 }}>
          {item.order_count} orders
        </Text>
      </View>
    </Pressable>
  );
}
