import React from 'react';
import { View, Text, Pressable, ScrollView, Alert, Modal } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import PayloadStatePanel from './PayloadStatePanel';
import ManifestKpiGrid from './ManifestKpiGrid';
import WorkflowSectionHeader from './WorkflowSectionHeader';
import { ExplainStatusBanner, type StatusExplain } from '../explainBanner';
import type { AppTheme } from '../theme';

export interface ManifestOrder {
  order_id: string;
  retailer_id?: string;
  payment_gateway?: string;
  amount?: number;
  delivery_lat?: number;
  delivery_lng?: number;
  [key: string]: unknown;
}

export interface ManifestItem {
  id: string;
  title?: string;
  label?: string;
  brand?: string;
  sku?: string;
  sku_name?: string;
  quantity?: number;
  verifiedQuantity?: number;
  scanned?: boolean;
  [key: string]: unknown;
}

export interface ManifestDetailPaneProps {
  selectedOrder: ManifestOrder | null;
  selectedOrderId: string | null;
  isIOS: boolean;
  theme: AppTheme;
  activeTruck: string | null;
  openReDispatch: (orderId: string) => void;
  manifestState: string | null;
  handleException: (orderId: string, reason: "OVERFLOW" | "DAMAGED" | "MANUAL") => void;
  exceptionLoading: string | null;
  setShowProductScanner: (show: boolean) => void;
  selectedManifest: ManifestItem[];
  toggleCheck: (itemId: string) => void;
  manifestId: string | null;
  handleSeal: () => void;
  allChecked: boolean;
  isSealing: boolean;
  setShowInjectOrder: (show: boolean) => void;
  sealExplain: StatusExplain | null;
  handleManifestSeal: () => void;
  isSealingManifest: boolean;
  isLoading: boolean;
  manifestVolume?: number;
  manifestMaxVolume?: number;
  manifestStopCount?: number;
  manifestRegionCode?: string;
  inboundDriverLat?: number | null;
  inboundDriverLng?: number | null;
  inboundLive?: boolean;
}

export const ManifestDetailPane: React.FC<ManifestDetailPaneProps> = ({
  selectedOrder,
  selectedOrderId,
  isIOS,
  theme: T,
  activeTruck,
  openReDispatch,
  manifestState,
  handleException,
  exceptionLoading,
  setShowProductScanner,
  selectedManifest,
  toggleCheck,
  manifestId,
  handleSeal,
  allChecked,
  isSealing,
  setShowInjectOrder,
  sealExplain,
  handleManifestSeal,
  isSealingManifest,
  isLoading,
  manifestVolume = 0,
  manifestMaxVolume = 0,
  manifestStopCount = 0,
  manifestRegionCode = '',
  inboundDriverLat = null,
  inboundDriverLng = null,
  inboundLive = false,
}) => {
  return (
    <View className="flex-1 flex-col">
      {inboundDriverLat != null && inboundDriverLng != null ? (
        <View
          className="px-8 py-3"
          style={{ borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}
        >
          <Text style={{ fontSize: 12, fontWeight: '600', color: T.colors.secondaryLabel }}>
            Inbound truck {inboundLive ? 'LIVE' : 'last known'}: {inboundDriverLat.toFixed(5)}, {inboundDriverLng.toFixed(5)}
          </Text>
        </View>
      ) : null}

      {/* Order header */}
      {selectedOrder ? (
        <>
          <View
            className="px-8 py-5 flex-row items-center justify-between"
            style={{ borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}
          >
            <View>
              <Text style={{ fontSize: 18, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0 }}>
                {selectedOrder.order_id}
              </Text>
              <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.3 }}>
                {selectedOrder.retailer_id} · {selectedOrder.payment_gateway} · {selectedOrder.amount?.toLocaleString()}
              </Text>
            </View>
            <View style={{ flexDirection: 'row', alignItems: 'center' }}>
              <View style={{
                borderWidth: isIOS ? 0.33 : 1,
                borderColor: T.colors.separator,
                borderRadius: T.radius.checkbox,
                paddingHorizontal: 12,
                paddingVertical: 6,
                backgroundColor: T.colors.fillTertiary,
              }}>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '600', fontSize: 11, color: T.colors.secondaryLabel, letterSpacing: 0.5 }}>
                  {activeTruck}
                </Text>
              </View>
              <Pressable
                onPress={() => selectedOrderId && openReDispatch(selectedOrderId)}
                style={{
                  marginLeft: 10,
                  flexDirection: 'row',
                  alignItems: 'center',
                  borderWidth: isIOS ? 0.33 : 1,
                  borderColor: T.colors.separator,
                  borderRadius: T.radius.checkbox,
                  paddingHorizontal: 12,
                  paddingVertical: 6,
                  backgroundColor: T.colors.fillTertiary,
                }}
              >
                <MaterialIcons name="swap-horiz" size={14} color={T.colors.secondaryLabel} style={{ marginRight: 4 }} />
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '600', fontSize: 11, color: T.colors.secondaryLabel, letterSpacing: 0.5 }}>
                  {isIOS ? 'Re-Dispatch' : 'RE-DISPATCH'}
                </Text>
              </Pressable>

              {/* Exception buttons — remove order from manifest */}
              {manifestState === 'LOADING' && selectedOrderId && (
                <View style={{ flexDirection: 'row', marginLeft: 8, gap: 4 }}>
                  {(['OVERFLOW', 'DAMAGED', 'MANUAL'] as const).map(reason => (
                    <Pressable
                      key={reason}
                      onPress={() => {
                        Alert.alert(
                          `Remove Order (${reason})`,
                          `Remove ${selectedOrderId.slice(0, 8)} from manifest? It will be re-injected with priority.`,
                          [
                            { text: 'Cancel', style: 'cancel' },
                            { text: 'Remove', style: 'destructive', onPress: () => handleException(selectedOrderId, reason) },
                          ]
                        );
                      }}
                      disabled={exceptionLoading === selectedOrderId}
                      style={({ pressed }) => ({
                        paddingHorizontal: 8,
                        paddingVertical: 6,
                        borderRadius: T.radius.checkbox,
                        borderWidth: isIOS ? 0.33 : 1,
                        borderColor: reason === 'DAMAGED' ? '#EF4444' : reason === 'OVERFLOW' ? '#F59E0B' : T.colors.separator,
                        backgroundColor: T.colors.fillTertiary,
                        opacity: pressed ? 0.75 : 1,
                        transform: [{ scale: pressed ? 0.95 : 1 }],
                      })}
                    >
                      <Text style={{
                        fontFamily: T.typography.mono.fontFamily,
                        fontWeight: '600',
                        fontSize: 9,
                        letterSpacing: 0.5,
                        color: reason === 'DAMAGED' ? '#EF4444' : reason === 'OVERFLOW' ? '#F59E0B' : T.colors.secondaryLabel,
                      }}>
                        {reason}
                      </Text>
                    </Pressable>
                  ))}
                </View>
              )}
            </View>
          </View>

          {/* Manifest checklist header */}
          <View style={{ paddingHorizontal: 32, paddingTop: 16, paddingBottom: 10, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 12 }}>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '900', fontSize: 12, color: T.colors.secondaryLabel, letterSpacing: 0.8 }}>
                ORDER_CHECKLIST
              </Text>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '700', fontSize: 11, color: allChecked ? T.colors.success : T.colors.accent }}>
                {selectedManifest.filter(i => i.scanned).length} / {selectedManifest.length} VERIFIED
              </Text>
            </View>
            <Pressable
              onPress={() => setShowProductScanner(true)}
              style={{
                paddingHorizontal: 12,
                paddingVertical: 6,
                borderRadius: T.radius.checkbox,
                borderWidth: isIOS ? 0.33 : 1,
                borderColor: T.colors.separator,
                backgroundColor: T.colors.fillTertiary,
                flexDirection: 'row',
                alignItems: 'center',
                gap: 4,
              }}
            >
              <MaterialIcons name="qr-code-scanner" size={14} color={T.colors.accent} />
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, fontWeight: '700', color: T.colors.accent }}>
                {isIOS ? 'SCAN' : 'SCAN'}
              </Text>
            </Pressable>
          </View>

          {/* Checklist items */}
          <ScrollView className="flex-1 px-8 py-2">
            {selectedManifest.map(item => (
              <Pressable
                key={item.id}
                onPress={() => toggleCheck(item.id)}
                style={({ pressed }) => ({
                  flexDirection: 'row' as const,
                  alignItems: 'center' as const,
                  paddingVertical: 14,
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
                  borderColor: item.scanned ? 'transparent' : T.colors.separator,
                  backgroundColor: item.scanned ? T.colors.accent : 'transparent',
                  marginRight: 16,
                  alignItems: 'center',
                  justifyContent: 'center',
                }}>
                  {item.scanned && (
                    <MaterialIcons name="check" size={14} color="#FFFFFF" />
                  )}
                </View>
                {/* Name + SKU */}
                <View style={{ flex: 1 }}>
                  <Text style={{
                    fontWeight: '600',
                    fontSize: 14,
                    color: item.scanned ? T.colors.tertiaryLabel : T.colors.label,
                    textDecorationLine: item.scanned ? 'line-through' : 'none',
                  }}>
                    {item.title || item.label || item.sku_name || item.sku || item.id}
                  </Text>
                  {(item.brand || item.sku) ? (
                    <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, marginTop: 2 }}>
                      {[item.brand, item.sku].filter(Boolean).join(' · ')}
                    </Text>
                  ) : null}
                </View>
                {/* Quantity */}
                <View style={{
                  paddingHorizontal: 8,
                  paddingVertical: 4,
                  borderRadius: 6,
                  backgroundColor: T.colors.fillTertiary,
                }}>
                  <Text style={{
                    fontFamily: T.typography.mono.fontFamily,
                    fontSize: 13,
                    fontWeight: '700',
                    color: item.scanned ? T.colors.tertiaryLabel : T.colors.label,
                  }}>
                    ×{item.quantity ?? 1}
                  </Text>
                </View>
              </Pressable>
            ))}
          </ScrollView>

          {/* Bottom Seal Bar */}
          <View
            className="px-8 py-5 flex-row items-center justify-between"
            style={{
              borderTopWidth: isIOS ? 0.33 : 1,
              borderTopColor: T.colors.separator,
              backgroundColor: T.colors.fillTertiary,
            }}
          >
            <Text style={{
              fontFamily: T.typography.mono.fontFamily,
              fontSize: 12,
              fontWeight: '600',
              color: allChecked ? T.colors.label : T.colors.tertiaryLabel,
            }}>
              {allChecked
                ? (isIOS ? 'All items verified' : 'ALL ITEMS VERIFIED')
                : `${selectedManifest.filter(i => i.scanned).length}/${selectedManifest.length} ${isIOS ? 'verified' : 'VERIFIED'}`}
            </Text>

            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 10 }}>
              {manifestId && manifestState === 'LOADING' && (
                <Pressable
                  onPress={() => setShowInjectOrder(true)}
                  style={({ pressed }) => ({
                    paddingHorizontal: 16,
                    paddingVertical: 12,
                    borderRadius: T.radius.button,
                    borderWidth: isIOS ? 0.33 : 1,
                    borderColor: T.colors.accent,
                    backgroundColor: T.colors.cardBackground,
                    opacity: pressed ? 0.82 : 1,
                    flexDirection: 'row',
                    alignItems: 'center',
                    gap: 4,
                  })}
                >
                  <MaterialIcons name="add" size={16} color={T.colors.accent} />
                  <Text style={{
                    fontFamily: T.typography.mono.fontFamily,
                    fontWeight: '700',
                    fontSize: 12,
                    color: T.colors.accent,
                  }}>
                    {isIOS ? 'Add Order' : 'ADD ORDER'}
                  </Text>
                </Pressable>
              )}

              <Pressable
                onPress={handleSeal}
                disabled={!allChecked || isSealing}
                style={({ pressed }) => ({
                  backgroundColor: allChecked ? T.colors.accent : T.colors.fillSecondary,
                  paddingHorizontal: 28,
                  paddingVertical: 12,
                  borderRadius: T.radius.button,
                  opacity: pressed ? 0.82 : 1,
                  transform: [{ scale: pressed ? 0.97 : 1 }],
                  ...T.shadow.card,
                })}
              >
                <Text style={{
                  fontWeight: '700',
                  fontSize: 13,
                  fontFamily: T.typography.mono.fontFamily,
                  letterSpacing: isIOS ? 0 : 0.5,
                  color: allChecked ? '#FFFFFF' : T.colors.tertiaryLabel,
                }}>
                  {isSealing
                    ? (isIOS ? 'Sealing...' : 'SEALING...')
                    : (isIOS ? 'Seal Order' : 'SEAL ORDER')}
                </Text>
              </Pressable>
            </View>
          </View>
        </>
      ) : (
        <ScrollView className="flex-1 p-8">
          {manifestId ? (
            <View style={{ marginBottom: 24 }}>
              <ManifestKpiGrid
                manifestId={manifestId}
                maxVolumeVu={manifestMaxVolume}
                regionCode={manifestRegionCode}
                state={manifestState || 'DRAFT'}
                stopCount={manifestStopCount}
                theme={T}
                totalVolumeVu={manifestVolume}
              />
            </View>
          ) : null}
          <View className="items-center justify-center p-8">
            <PayloadStatePanel
              compact
              message={isIOS ? 'Choose an order from the sidebar to review its checklist.' : 'CHOOSE AN ORDER FROM THE SIDEBAR TO REVIEW ITS CHECKLIST.'}
              theme={T}
              title={isLoading ? (isIOS ? 'Fetching manifest...' : 'FETCHING MANIFEST...') : (isIOS ? 'Select order from manifest' : 'SELECT ORDER FROM MANIFEST')}
              variant="manifest"
            />
          </View>
        </ScrollView>
      )}
    </View>
  );
};
