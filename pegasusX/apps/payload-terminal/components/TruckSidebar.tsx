import React from 'react';
import { View, Text, Pressable } from 'react-native';
import { isIOS, useT } from '../theme';
import PayloadStatePanel from './PayloadStatePanel';

export interface Truck {
  id: string;
  label: string;
  license_plate: string;
  vehicle_class: string;
}

interface TruckSidebarProps {
  trucks: Truck[];
  activeTruck: string | null;
  setActiveTruck: (id: string) => void;
  isLoadingTrucks: boolean;
  variant?: 'selector' | 'sidebar';
  tx: (key: string) => string;
}

export default function TruckSidebar({
  trucks,
  activeTruck,
  setActiveTruck,
  isLoadingTrucks,
  variant = 'sidebar',
  tx,
}: TruckSidebarProps) {
  const T = useT();

  if (variant === 'selector') {
    return (
      <View className="flex-1 items-center justify-center p-12">
        {isLoadingTrucks ? (
          <PayloadStatePanel
            theme={T}
            variant="truck"
            title={isIOS ? 'Loading vehicles...' : 'LOADING VEHICLES...'}
            message={isIOS ? 'Refreshing supplier fleet availability for this shift.' : 'REFRESHING SUPPLIER FLEET AVAILABILITY FOR THIS SHIFT.'}
          />
        ) : trucks.length === 0 ? (
          <PayloadStatePanel
            theme={T}
            variant="truck"
            title={tx('payload.vehicle.none_available')}
            message={isIOS ? 'No payload vehicle is currently ready for assignment.' : 'NO PAYLOAD VEHICLE IS CURRENTLY READY FOR ASSIGNMENT.'}
            tone="warning"
          />
        ) : (
          <>
            <Text style={{ fontSize: 13, fontWeight: '500', color: T.colors.tertiaryLabel, marginBottom: 32, letterSpacing: 0.3 }}>
              {tx('payload.vehicle.select_target')}
            </Text>
            <View className="flex-row gap-4">
              {trucks.map(truck => (
                <Pressable
                  key={truck.id}
                  onPress={() => setActiveTruck(truck.id)}
                  style={({ pressed }) => ({
                    borderWidth: isIOS ? 0.33 : 1,
                    borderColor: T.colors.separator,
                    backgroundColor: T.colors.cardBackground,
                    paddingHorizontal: 40,
                    paddingVertical: 32,
                    alignItems: 'center' as const,
                    borderRadius: T.radius.card,
                    ...T.shadow.card,
                    opacity: pressed ? 0.82 : 1,
                    transform: [{ scale: pressed ? 0.96 : 1 }],
                  })}
                >
                  <Text style={{ fontSize: 22, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 1 }}>
                    {truck.label}
                  </Text>
                  {truck.license_plate ? (
                    <Text style={{ fontSize: 11, fontFamily: T.typography.mono.fontFamily, color: T.colors.tertiaryLabel, marginTop: 6, letterSpacing: 0.5 }}>
                      {truck.license_plate}
                    </Text>
                  ) : null}
                  <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.3 }}>
                    {truck.vehicle_class}
                  </Text>
                </Pressable>
              ))}
            </View>
            <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 40, letterSpacing: 0.3 }}>
              {isIOS ? 'Select target vehicle' : 'SELECT TARGET VEHICLE'}
            </Text>
          </>
        )}
      </View>
    );
  }

  // sidebar variant
  return (
    <View style={{ flexDirection: 'row', borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
      {trucks.map(truck => (
        <Pressable
          key={truck.id}
          onPress={() => setActiveTruck(truck.id)}
          style={{
            flex: 1,
            paddingVertical: 10,
            alignItems: 'center',
            backgroundColor: activeTruck === truck.id ? T.colors.sidebarActive : 'transparent',
            borderRadius: activeTruck === truck.id ? 8 : 0,
            margin: activeTruck === truck.id ? 4 : 0,
          }}
        >
          <Text style={{
            fontWeight: '700',
            fontSize: 11,
            letterSpacing: 0.5,
            color: activeTruck === truck.id ? T.colors.sidebarActiveText : T.colors.sidebarSecondary,
          }}>
            {truck.label}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}
