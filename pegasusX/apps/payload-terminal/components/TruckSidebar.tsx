import React from 'react';
import { View, Text, Pressable } from 'react-native';
import { isIOS, useT } from '../theme';
import PayloadStatePanel from './PayloadStatePanel';
import {
  BOARD_MANIFEST_STATES,
  groupBoardColumns,
  unassignedTrucks,
  type BoardTruck,
} from '../utils/manifestBoard';

export type Truck = BoardTruck;

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
            <Text style={{ fontSize: 13, fontWeight: '500', color: T.colors.tertiaryLabel, marginBottom: 16, letterSpacing: 0.3 }}>
              {isIOS ? 'Board by manifest state' : 'BOARD BY MANIFEST STATE'}
            </Text>
            <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 16, justifyContent: 'center', paddingHorizontal: 16 }}>
              {BOARD_MANIFEST_STATES.map((state) => {
                const column = groupBoardColumns(trucks)[state];
                return (
                  <View key={state} style={{ minWidth: 180, maxWidth: 260 }}>
                    <Text style={{ fontSize: 11, fontWeight: '700', color: T.colors.tertiaryLabel, letterSpacing: 0.6, marginBottom: 8 }}>
                      {state} · {column.length === 0 ? 'empty' : String(column.length)}
                    </Text>
                    {column.length === 0 ? (
                      <Text style={{ fontSize: 11, color: T.colors.tertiaryLabel }}>
                        {isIOS ? `No ${state.toLowerCase()} manifests` : `NO ${state} MANIFESTS`}
                      </Text>
                    ) : column.map((truck) => (
                      <Pressable
                        key={truck.id}
                        onPress={() => setActiveTruck(truck.id)}
                        style={({ pressed }) => ({
                          borderWidth: isIOS ? 0.33 : 1,
                          borderColor: T.colors.separator,
                          backgroundColor: T.colors.cardBackground,
                          paddingHorizontal: 16,
                          paddingVertical: 14,
                          marginBottom: 8,
                          borderRadius: T.radius.card,
                          opacity: pressed ? 0.82 : 1,
                        })}
                      >
                        <Text style={{ fontSize: 16, fontWeight: '700', color: T.colors.label }}>{truck.label}</Text>
                        {truck.license_plate ? (
                          <Text style={{ fontSize: 11, fontFamily: T.typography.mono.fontFamily, color: T.colors.tertiaryLabel, marginTop: 4 }}>
                            {truck.license_plate}
                          </Text>
                        ) : null}
                        {truck.max_volume_vu ? (
                          <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4 }}>
                            {truck.used_volume_vu ?? 0}/{truck.max_volume_vu} VU
                          </Text>
                        ) : null}
                      </Pressable>
                    ))}
                  </View>
                );
              })}
            </View>
            {unassignedTrucks(trucks).length > 0 ? (
              <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 24 }}>
                {isIOS ? 'Some vehicles have no open manifest' : 'SOME VEHICLES HAVE NO OPEN MANIFEST'}
              </Text>
            ) : null}
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
