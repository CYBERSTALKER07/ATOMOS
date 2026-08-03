import React from 'react';
import { View, Text, Pressable, ScrollView, Alert, Modal } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { PayloadStatePanel } from './PayloadStatePanel';
import { WorkflowSectionHeader } from './WorkflowSectionHeader';
import { ExplainStatusBanner } from './ExplainStatusBanner';

export interface ManifestDetailPaneProps {
  selectedOrder: any | null;
  selectedOrderId: string | null;
  isIOS: boolean;
  theme: any;
  activeTruck: string | null;
  openReDispatch: (orderId: string) => void;
  manifestState: string | null;
  handleException: (orderId: string, reason: string) => void;
  exceptionLoading: string | null;
  setShowProductScanner: (show: boolean) => void;
  selectedManifest: any[];
  toggleCheck: (itemId: string) => void;
  manifestId: string | null;
  handleSeal: () => void;
  allChecked: boolean;
  isSealing: boolean;
  setShowInjectOrder: (show: boolean) => void;
  sealExplain: string | null;
  handleManifestSeal: () => void;
  isSealingManifest: boolean;
  isLoading: boolean;
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
}) => {
  return (
      <View className="flex-1 flex-col">
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

              {/* LEO: Exception buttons — remove order from manifest */}
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

            {/* Manifest checklist */}
            <View style={{ paddingHorizontal: 32, paddingTop: 16, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
              <WorkflowSectionHeader
                subtitle={isIOS ? 'Tap each line to confirm load verification.' : 'TAP EACH LINE TO CONFIRM LOAD VERIFICATION.'}
                theme={T}
                title="Load checklist"
              />
              <Pressable
                onPress={() => setShowProductScanner(true)}
                style={{
                  paddingHorizontal: 12,
                  paddingVertical: 8,
                  borderRadius: T.radius.button,
                  borderWidth: 1,
                  borderColor: T.colors.accent,
                }}
              >
                <Text style={{ fontSize: 11, fontWeight: '700', color: T.colors.accent }}>
                  {isIOS ? 'Scan product' : 'SCAN PRODUCT'}
                </Text>
              </Pressable>
            </View>
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

            {/* Seal button — per-order (legacy) + manifest-level (LEO) */}
            <View style={{ paddingHorizontal: 32, paddingVertical: 20, borderTopWidth: isIOS ? 0.33 : 1, borderTopColor: T.colors.separator }}>
              <WorkflowSectionHeader
                subtitle={manifestId ? (isIOS ? 'Inject orders or seal the manifest when loading is complete.' : 'INJECT ORDERS OR SEAL THE MANIFEST WHEN LOADING IS COMPLETE.') : (isIOS ? 'Seal each order after checklist verification.' : 'SEAL EACH ORDER AFTER CHECKLIST VERIFICATION.')}
                theme={T}
                title={manifestId ? 'Manifest workflow' : 'Order seal'}
              />
              <View style={{ marginTop: 12 }}>
              {/* Per-order seal (legacy — when no manifest entity exists) */}
              {!manifestId && (
                <Pressable
                  onPress={handleSeal}
                  disabled={!allChecked || isSealing}
                  style={({ pressed }) => ({
                    paddingVertical: 16,
                    alignItems: 'center' as const,
                    backgroundColor: allChecked && !isSealing ? T.colors.accent : T.colors.fillSecondary,
                    borderRadius: T.radius.button,
                    opacity: pressed ? 0.82 : 1,
                    transform: [{ scale: pressed ? 0.97 : 1 }],
                  })}
                >
                  <Text style={{
                    fontWeight: '600',
                    fontSize: 14,
                    letterSpacing: isIOS ? 0.3 : 1,
                    color: allChecked && !isSealing ? '#FFFFFF' : T.colors.tertiaryLabel,
                  }}>
                    {isSealing ? (isIOS ? 'Sealing...' : 'SEALING...') : (isIOS ? 'Mark as Loaded' : 'MARK AS LOADED')}
                  </Text>
                </Pressable>
              )}
              {/* Manifest-level seal (LEO — slide to seal entire manifest) */}
              {manifestId && manifestState === 'LOADING' && (
                <View style={{ gap: 10 }}>
                  {/* Inject order button */}
                  <Pressable
                    onPress={() => setShowInjectOrder(true)}
                    style={{
                      paddingVertical: 14,
                      alignItems: 'center',
                      backgroundColor: T.colors.fillSecondary,
                      borderRadius: T.radius.button,
                      borderWidth: 1,
                      borderColor: T.colors.accent,
                      flexDirection: 'row',
                      justifyContent: 'center',
                      gap: 8,
                    }}
                  >
                    <MaterialIcons name="add-circle-outline" size={18} color={T.colors.accent} />
                    <Text style={{
                      fontWeight: '600',
                      fontSize: 13,
                      letterSpacing: isIOS ? 0.3 : 1,
                      color: T.colors.accent,
                    }}>
                      {isIOS ? 'Add Order' : 'ADD ORDER'}
                    </Text>
                  </Pressable>
                  <ExplainStatusBanner explain={sealExplain} />
                  {/* Seal manifest button */}
                  <Pressable
                    onPress={handleManifestSeal}
                    disabled={isSealingManifest}
                    style={{
                      paddingVertical: 18,
                      alignItems: 'center',
                      backgroundColor: isSealingManifest ? T.colors.fillSecondary : '#16A34A',
                      borderRadius: T.radius.button,
                      flexDirection: 'row',
                      justifyContent: 'center',
                      gap: 8,
                    }}
                  >
                    <MaterialIcons name="verified" size={18} color="#FFFFFF" />
                    <Text style={{
                      fontWeight: '700',
                      fontSize: 14,
                      letterSpacing: isIOS ? 0.3 : 1.2,
                      color: '#FFFFFF',
                    }}>
                      {isSealingManifest ? (isIOS ? 'Sealing Manifest...' : 'SEALING MANIFEST...') : (isIOS ? 'Seal Manifest' : 'SEAL MANIFEST')}
                    </Text>
                  </Pressable>
                </View>
              )}
              {manifestId && manifestState === 'SEALED' && (
                <View style={{ paddingVertical: 14, alignItems: 'center', backgroundColor: T.colors.fillTertiary, borderRadius: T.radius.button }}>
                  <Text style={{ fontWeight: '600', fontSize: 13, color: '#16A34A', letterSpacing: 0.5 }}>
                    {isIOS ? 'Manifest Sealed — Route Finalized' : 'MANIFEST SEALED — ROUTE FINALIZED'}
                  </Text>
                </View>
              )}
              </View>
            </View>
          </>
        ) : (
          <View className="flex-1 items-center justify-center p-8">
            <PayloadStatePanel
              compact
              message={isIOS ? 'Choose an order from the sidebar to review its checklist.' : 'CHOOSE AN ORDER FROM THE SIDEBAR TO REVIEW ITS CHECKLIST.'}
              theme={T}
              title={isLoading ? (isIOS ? 'Fetching manifest...' : 'FETCHING MANIFEST...') : (isIOS ? 'Select order from manifest' : 'SELECT ORDER FROM MANIFEST')}
              variant="manifest"
            />
          </View>
        )}
      </View>
  );
};
