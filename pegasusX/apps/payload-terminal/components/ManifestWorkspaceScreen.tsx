import type { ReactNode } from 'react';
import { Alert, FlatList, Modal, ScrollView, Text, TextInput, View } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { CameraView } from 'expo-camera';

import { ExplainStatusBanner } from '../explainBanner';
import { isIOS, type AppTheme } from '../theme';
import type { useManifestActions } from '../hooks/useManifestActions';
import type { useManifestData } from '../hooks/useManifestData';
import type { useManifestExceptions } from '../hooks/useManifestExceptions';
import type { useNotifications } from '../hooks/useNotifications';
import type { useOfflineQueue } from '../hooks/useOfflineQueue';
import type { useReDispatch } from '../hooks/useReDispatch';
import ConnectionStrip from './ConnectionStrip';
import ManifestKpiGrid from './ManifestKpiGrid';
import { ManifestDetailPane } from './ManifestDetailPane';
import NotificationsSheet from './NotificationsSheet';
import ExceptionsSheet from './ExceptionsSheet';
import OrderChecklist from './OrderChecklist';
import PayloadStatePanel from './PayloadStatePanel';
import Pressable from './Pressable';
import RecommendationCard from './RecommendationCard';
import StatusBadge from './StatusBadge';
import TruckSidebar from './TruckSidebar';
import WorkflowSectionHeader from './WorkflowSectionHeader';

// ─── Render: MANIFEST VIEW ────────────────────────────────────────────────────

type ManifestWorkspaceScreenProps = {
  theme: AppTheme;
  tx: (key: string) => string;
  onShowInbound: () => void;
  policyBanner: ReactNode;
  toast: ReactNode;
  data: ReturnType<typeof useManifestData>;
  actions: ReturnType<typeof useManifestActions>;
  redispatch: ReturnType<typeof useReDispatch>;
  notif: ReturnType<typeof useNotifications>;
  exceptions: ReturnType<typeof useManifestExceptions>;
  queue: ReturnType<typeof useOfflineQueue>;
};

export default function ManifestWorkspaceScreen({
  theme: T,
  tx,
  onShowInbound,
  policyBanner,
  toast,
  data,
  actions,
  redispatch,
  notif,
  exceptions,
  queue,
}: ManifestWorkspaceScreenProps) {
  const {
    activeTruck,
    allChecked,
    batchReadyManifestIds,
    deliveryLabelsByOrder,
    handleTruckSelect,
    inboundDriverLat,
    inboundDriverLng,
    inboundLive,
    isInjecting,
    isLoading,
    isLoadingTrucks,
    isOnline,
    isSealingManifest,
    isStartingLoad,
    manifestId,
    manifestMaxVolume,
    manifestRegionCode,
    manifestState,
    manifestStopCount,
    manifestVolume,
    orders,
    sealedOrderIds,
    selectedManifest,
    selectedOrder,
    selectedOrderId,
    setSelectedOrderId,
    trucks,
  } = data;
  const {
    batchSealing,
    batchSealFailures,
    exceptionLoading,
    handleException,
    handleFinalizeBatchSeal,
    handleInjectOrder,
    handleManifestSeal,
    handleProductBarcodeScan,
    handleSeal,
    handleStartLoading,
    injectOrderId,
    isSealing,
    sealExplain,
    setInjectOrderId,
    setShowInjectOrder,
    setShowInjectScanner,
    setShowProductScanner,
    showInjectOrder,
    showInjectScanner,
    showProductScanner,
    toggleCheck,
  } = actions;
  const {
    handleReassign,
    isLoadingRecs,
    isReassigning,
    openReDispatch,
    recommendations,
    reDispatchOrderId,
    reDispatchRetailer,
    reDispatchVolume,
    setReDispatchOrderId,
    setShowReDispatch,
    showReDispatch,
  } = redispatch;
  const {
    markAllNotifsRead,
    markNotifRead,
    notifications,
    setShowNotifPanel,
    showNotifPanel,
    unreadCount,
  } = notif;
  const {
    loadManifestExceptions,
    loadingExceptions,
    manifestExceptions,
    setShowExceptionsPanel,
    showExceptionsPanel,
  } = exceptions;
  const { offlineQueue } = queue;

  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background, flexDirection: 'column' }}>
      {policyBanner}
      <View style={{ flex: 1, flexDirection: 'row' }}>

      {/* ── Left pane: Shop list ─────────────────────────────────────────── */}
      <View style={{ width: 288, backgroundColor: T.colors.sidebarBackground, flexDirection: 'column' }}>
        {/* Header */}
        <View style={{ paddingHorizontal: 24, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator, flexDirection: 'row', alignItems: 'center' }}>
          <View style={{ flex: 1 }}>
            <Text style={{ color: T.colors.sidebarLabel, fontWeight: '700', fontSize: 13, letterSpacing: 0.3, marginBottom: 2 }}>
              {isIOS ? 'Payload Terminal' : 'PAYLOAD TERMINAL'}
            </Text>
            <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 11 }}>
              {activeTruck}
            </Text>
            <ConnectionStrip isOnline={isOnline} queuedCount={offlineQueue.length} theme={T} />
          </View>
          <Pressable
            onPress={onShowInbound}
            style={{ padding: 6, marginRight: 4 }}
          >
            <MaterialIcons name="undo" size={20} color={T.colors.sidebarLabel} />
          </Pressable>
          <Pressable
            onPress={() => {
              setShowExceptionsPanel(true);
              void loadManifestExceptions();
            }}
            style={{ padding: 6, marginRight: 4 }}
          >
            <MaterialIcons name="report-problem" size={20} color={T.colors.sidebarLabel} />
            {manifestExceptions.length > 0 ? (
              <View style={{ position: 'absolute', top: 2, right: 2, backgroundColor: '#F59E0B', borderRadius: 8, minWidth: 16, height: 16, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#FFF', fontSize: 9, fontWeight: '700' }}>
                  {manifestExceptions.length > 99 ? '99+' : manifestExceptions.length}
                </Text>
              </View>
            ) : null}
          </Pressable>
          <Pressable onPress={() => setShowNotifPanel(true)} style={{ padding: 6 }}>
            <MaterialIcons name="notifications" size={20} color={T.colors.sidebarLabel} />
            {unreadCount > 0 && (
              <View style={{ position: 'absolute', top: 2, right: 2, backgroundColor: '#EF4444', borderRadius: 8, minWidth: 16, height: 16, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#FFF', fontSize: 9, fontWeight: '700' }}>{unreadCount > 99 ? '99+' : unreadCount}</Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* Truck toggle bar */}
        <TruckSidebar
          variant="sidebar"
          trucks={trucks}
          activeTruck={activeTruck}
          setActiveTruck={handleTruckSelect}
          isLoadingTrucks={isLoadingTrucks}
          tx={tx}
        />

        {batchReadyManifestIds.length > 1 && (
          <View style={{
            paddingHorizontal: 12,
            paddingVertical: 10,
            borderBottomWidth: 0.5,
            borderBottomColor: T.colors.sidebarSeparator,
            backgroundColor: `${T.colors.accent}12`,
          }}>
            <WorkflowSectionHeader
              onDark
              subtitle={`${batchReadyManifestIds.length} trucks ready to finalize`}
              theme={T}
              title="Batch seal"
            />
            <Pressable
              onPress={handleFinalizeBatchSeal}
              disabled={batchSealing}
              style={{
                paddingVertical: 10,
                alignItems: 'center',
                backgroundColor: batchSealing ? T.colors.fillSecondary : T.colors.accent,
                borderRadius: T.radius.button,
              }}
            >
              <Text style={{ fontWeight: '700', fontSize: 11, color: batchSealing ? T.colors.tertiaryLabel : '#FFFFFF' }}>
                {batchSealing ? (isIOS ? 'Finalizing…' : 'FINALIZING…') : (isIOS ? 'Seal all trucks' : 'SEAL ALL TRUCKS')}
              </Text>
            </Pressable>
            {batchSealFailures.map((row, idx) => (
              <View key={`${row.manifest_id ?? 'manifest'}-${idx}`} style={{ marginTop: 8 }}>
                <Text style={{ fontSize: 11, color: T.colors.secondaryLabel, marginBottom: 4 }}>
                  {(row.manifest_id ?? 'manifest').slice(0, 12)} · {row.status ?? 'failed'}
                </Text>
                {row.explain ? <ExplainStatusBanner explain={row.explain} /> : null}
              </View>
            ))}
          </View>
        )}

        {/* LEO: Volume Progress Bar + Manifest State */}
        {manifestId && (
          <View style={{ paddingHorizontal: 16, paddingVertical: 10, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
            <ManifestKpiGrid
              compact
              manifestId={manifestId}
              maxVolumeVu={manifestMaxVolume}
              regionCode={manifestRegionCode}
              state={manifestState}
              stopCount={manifestStopCount}
              theme={T}
              totalVolumeVu={manifestVolume}
            />
            {manifestState === 'DRAFT' && (
              <Pressable
                onPress={handleStartLoading}
                disabled={isStartingLoad}
                style={{
                  marginTop: 10,
                  paddingVertical: 10,
                  alignItems: 'center',
                  backgroundColor: isStartingLoad ? T.colors.fillSecondary : T.colors.accent,
                  borderRadius: T.radius.button,
                }}
              >
                <Text style={{ fontWeight: '600', fontSize: 12, letterSpacing: 0.5, color: isStartingLoad ? T.colors.tertiaryLabel : '#FFFFFF' }}>
                  {isStartingLoad ? (isIOS ? 'Starting...' : 'STARTING...') : (isIOS ? 'Start Loading' : 'START LOADING')}
                </Text>
              </Pressable>
            )}
          </View>
        )}

        {/* Order list */}
        <ScrollView>
          {isLoading ? (
            <View className="p-6 items-center">
              <PayloadStatePanel
                theme={T}
                variant="manifest"
                title={isIOS ? 'Fetching manifest...' : 'FETCHING MANIFEST...'}
                message={isIOS ? 'Loading the active checklist for this truck.' : 'LOADING THE ACTIVE CHECKLIST FOR THIS TRUCK.'}
                compact
              />
            </View>
          ) : orders.length === 0 ? (
            <View className="p-6 items-center">
              <PayloadStatePanel
                theme={T}
                variant="manifest"
                title={isIOS ? 'No pending orders' : 'NO PENDING ORDERS'}
                message={isIOS ? 'This truck has no checklist items waiting to load.' : 'THIS TRUCK HAS NO CHECKLIST ITEMS WAITING TO LOAD.'}
                compact
              />
            </View>
          ) : (
            orders.map(order => {
              const isSealed = sealedOrderIds.has(order.order_id);
              const isActive = order.order_id === selectedOrderId;
              return (
                <Pressable
                  key={order.order_id}
                  onPress={() => !isSealed && setSelectedOrderId(order.order_id)}
                  onLongPress={() => {
                    if (!isSealed) {
                      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
                      openReDispatch(order.order_id);
                    }
                  }}
                  delayLongPress={500}
                  style={{
                    paddingHorizontal: 24,
                    paddingVertical: 14,
                    borderBottomWidth: 0.5,
                    borderBottomColor: T.colors.sidebarSeparator,
                    flexDirection: 'row',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    backgroundColor: isActive ? T.colors.sidebarActive : 'transparent',
                    borderRadius: isActive ? (isIOS ? 10 : 16) : 0,
                    marginHorizontal: isActive ? 8 : 0,
                    marginVertical: isActive ? 2 : 0,
                  }}
                >
                  <View>
                    <Text style={{ fontWeight: '600', fontSize: 13, color: isActive ? T.colors.sidebarActiveText : isSealed ? T.colors.sidebarSecondary : T.colors.sidebarLabel }}>
                      {order.order_id}
                    </Text>
                    <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, marginTop: 2, color: isActive ? (isIOS ? 'rgba(0,0,0,0.5)' : T.colors.sidebarActiveText) : T.colors.sidebarSecondary }}>
                      {(order.delivery_expectation?.target_label || deliveryLabelsByOrder[order.order_id]) ?? order.retailer_id}
                    </Text>
                  </View>
                  {isSealed && (
                    <StatusBadge compact label={isIOS ? 'Cleared' : 'CLEARED'} theme={T} tone="success" />
                  )}
                </Pressable>
              );
            })
          )}
        </ScrollView>
      </View>

      {/* ── Right pane: Manifest detail ──────────────────────────────────── */}
      <ManifestDetailPane
        selectedOrder={selectedOrder}
        selectedOrderId={selectedOrderId}
        isIOS={isIOS}
        theme={T}
        activeTruck={activeTruck}
        openReDispatch={openReDispatch}
        manifestState={manifestState}
        handleException={handleException}
        exceptionLoading={exceptionLoading}
        setShowProductScanner={setShowProductScanner}
        selectedManifest={selectedManifest}
        toggleCheck={toggleCheck}
        manifestId={manifestId}
        handleSeal={handleSeal}
        allChecked={allChecked}
        isSealing={isSealing}
        setShowInjectOrder={setShowInjectOrder}
        sealExplain={sealExplain}
        handleManifestSeal={handleManifestSeal}
        isSealingManifest={isSealingManifest}
        isLoading={isLoading}
        inboundDriverLat={inboundDriverLat}
        inboundDriverLng={inboundDriverLng}
        inboundLive={inboundLive}
      />
      {false && (
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
                  {selectedOrder?.order_id}
                </Text>
                <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.3 }}>
                  {selectedOrder?.retailer_id} · {selectedOrder?.payment_gateway} · {selectedOrder?.amount?.toLocaleString()}
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
                onPress={() => selectedOrderId && openReDispatch(selectedOrderId!)}
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
                          `Remove ${selectedOrderId!.slice(0, 8)} from manifest? It will be re-injected with priority.`,
                          [
                            { text: 'Cancel', style: 'cancel' },
                            { text: 'Remove', style: 'destructive', onPress: () => handleException(selectedOrderId!, reason as "OVERFLOW" | "DAMAGED" | "MANUAL") },
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
            <OrderChecklist
              selectedManifest={selectedManifest}
              theme={T}
              toggleCheck={toggleCheck}
            />

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
      )}

      {/* ── Inject Order Modal ────────────────────────────────────────── */}
      <Modal visible={showInjectOrder} transparent animationType="fade" onRequestClose={() => setShowInjectOrder(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 400, backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden' }}>
            <View style={{ paddingHorizontal: 24, paddingVertical: 16, borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}>
              <Text style={{ fontWeight: '700', fontSize: 17, color: T.colors.label }}>
                {isIOS ? 'Add Order to Manifest' : 'ADD ORDER TO MANIFEST'}
              </Text>
              <Text style={{ fontSize: 12, color: T.colors.secondaryLabel, marginTop: 4 }}>
                Scan an order label or enter the order ID to inject into the active loading session.
              </Text>
            </View>
            <View style={{ padding: 24, gap: 16 }}>
              <Pressable
                onPress={() => setShowInjectScanner((v) => !v)}
                style={{
                  paddingVertical: 10,
                  alignItems: 'center',
                  borderRadius: T.radius.button,
                  borderWidth: 1,
                  borderColor: T.colors.accent,
                }}
              >
                <Text style={{ fontWeight: '600', fontSize: 13, color: T.colors.accent }}>
                  {showInjectScanner ? (isIOS ? 'Hide scanner' : 'HIDE SCANNER') : (isIOS ? 'Scan order label' : 'SCAN ORDER LABEL')}
                </Text>
              </Pressable>
              {showInjectScanner ? (
                <CameraView
                  style={{ height: 160, borderRadius: 12, overflow: 'hidden' }}
                  barcodeScannerSettings={{ barcodeTypes: ['ean13', 'ean8', 'code128', 'qr'] }}
                  onBarcodeScanned={({ data }) => {
                    setInjectOrderId(data.trim());
                    setShowInjectScanner(false);
                  }}
                />
              ) : null}
              <TextInput
                value={injectOrderId}
                onChangeText={setInjectOrderId}
                placeholder="Order ID (UUID)"
                placeholderTextColor={T.colors.tertiaryLabel}
                autoCapitalize="none"
                autoCorrect={false}
                style={{
                  fontFamily: T.typography.mono.fontFamily,
                  fontSize: 14,
                  color: T.colors.label,
                  backgroundColor: T.colors.fillTertiary,
                  borderRadius: (T.radius as any).input || 8,
                  paddingHorizontal: 16,
                  paddingVertical: 12,
                  borderWidth: 1,
                  borderColor: T.colors.separator,
                }}
              />
              {!isOnline && (
                <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6, paddingVertical: 4 }}>
                  <MaterialIcons name="cloud-off" size={14} color="#F59E0B" />
                  <Text style={{ fontSize: 11, color: '#F59E0B' }}>Offline — action will be queued</Text>
                </View>
              )}
              <View style={{ flexDirection: 'row', gap: 12 }}>
                <Pressable
                  onPress={() => { setShowInjectOrder(false); setInjectOrderId(''); }}
                  style={{ flex: 1, paddingVertical: 14, alignItems: 'center', backgroundColor: T.colors.fillSecondary, borderRadius: T.radius.button }}
                >
                  <Text style={{ fontWeight: '600', fontSize: 14, color: T.colors.secondaryLabel }}>Cancel</Text>
                </Pressable>
                <Pressable
                  onPress={handleInjectOrder}
                  disabled={!injectOrderId.trim() || isInjecting}
                  style={{
                    flex: 1,
                    paddingVertical: 14,
                    alignItems: 'center',
                    backgroundColor: injectOrderId.trim() && !isInjecting ? T.colors.accent : T.colors.fillSecondary,
                    borderRadius: T.radius.button,
                  }}
                >
                  <Text style={{ fontWeight: '700', fontSize: 14, color: injectOrderId.trim() && !isInjecting ? '#FFFFFF' : T.colors.tertiaryLabel }}>
                    {isInjecting ? 'Adding...' : 'Add Order'}
                  </Text>
                </Pressable>
              </View>
            </View>
          </View>
        </View>
      </Modal>

      <Modal visible={showProductScanner} transparent animationType="fade" onRequestClose={() => setShowProductScanner(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 420, backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden', padding: 16, gap: 12 }}>
            <Text style={{ fontWeight: '700', fontSize: 16, color: T.colors.label }}>
              {isIOS ? 'Scan product EAN' : 'SCAN PRODUCT EAN'}
            </Text>
            <CameraView
              style={{ height: 200, borderRadius: 12, overflow: 'hidden' }}
              barcodeScannerSettings={{ barcodeTypes: ['ean13', 'ean8'] }}
              onBarcodeScanned={({ data }) => { void handleProductBarcodeScan(data); }}
            />
            <Pressable onPress={() => setShowProductScanner(false)} style={{ paddingVertical: 12, alignItems: 'center' }}>
              <Text style={{ color: T.colors.secondaryLabel }}>{isIOS ? 'Close' : 'CLOSE'}</Text>
            </Pressable>
          </View>
        </View>
      </Modal>

      {/* ── Offline Queue Indicator ────────────────────────────────────── */}
      {offlineQueue.length > 0 && (
        <View style={{
          position: 'absolute', bottom: 12, left: 12,
          flexDirection: 'row', alignItems: 'center', gap: 6,
          backgroundColor: 'rgba(245, 158, 11, 0.95)', paddingHorizontal: 12, paddingVertical: 6,
          borderRadius: 8,
        }}>
          <MaterialIcons name="cloud-queue" size={14} color="#FFFFFF" />
          <Text style={{ fontSize: 11, fontWeight: '600', color: '#FFFFFF' }}>
            {offlineQueue.length} queued action{offlineQueue.length > 1 ? 's' : ''} pending sync
          </Text>
        </View>
      )}

      {/* ── Re-Dispatch Modal ────────────────────────────────────────── */}
      <Modal visible={showReDispatch} transparent animationType="fade" onRequestClose={() => { setShowReDispatch(false); setReDispatchOrderId(null); }}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 520, maxHeight: '85%', backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden' }}>
            {/* Header */}
            <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 24, paddingVertical: 16, borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}>
              <View style={{ flex: 1 }}>
                <Text style={{ fontWeight: '700', fontSize: 17, color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0 }}>
                  {isIOS ? 'Re-Dispatch Order' : 'RE-DISPATCH ORDER'}
                </Text>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.5 }}>
                  {reDispatchOrderId}
                </Text>
                {reDispatchRetailer ? (
                  <Text style={{ fontSize: 12, color: T.colors.secondaryLabel, marginTop: 2 }}>
                    {reDispatchRetailer} · {reDispatchVolume.toFixed(1)} VU
                  </Text>
                ) : null}
              </View>
              <Pressable onPress={() => { setShowReDispatch(false); setReDispatchOrderId(null); }} style={{ padding: 8 }}>
                <MaterialIcons name="close" size={22} color={T.colors.tertiaryLabel} />
              </Pressable>
            </View>

            {/* Recommendation list */}
            {isLoadingRecs ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <PayloadStatePanel
                  theme={T}
                  variant="dispatch"
                  title={isIOS ? 'Analyzing fleet positions...' : 'ANALYZING FLEET POSITIONS...'}
                  message={isIOS ? 'Scoring nearby trucks for the best reassignment path.' : 'SCORING NEARBY TRUCKS FOR THE BEST REASSIGNMENT PATH.'}
                  compact
                />
              </View>
            ) : recommendations.length === 0 ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <PayloadStatePanel
                  theme={T}
                  variant="dispatch"
                  title={isIOS ? 'No available trucks found' : 'NO AVAILABLE TRUCKS FOUND'}
                  message={isIOS ? 'No nearby fleet target can accept this order right now.' : 'NO NEARBY FLEET TARGET CAN ACCEPT THIS ORDER RIGHT NOW.'}
                  compact
                  tone="warning"
                />
              </View>
            ) : (
              <FlatList
                data={recommendations}
                keyExtractor={item => item.driver_id}
                style={{ maxHeight: 400 }}
                renderItem={({ item, index }) => (
                  <RecommendationCard
                    item={item}
                    index={index}
                    reDispatchVolume={reDispatchVolume}
                    isReassigning={isReassigning}
                    theme={T}
                    onReassign={handleReassign}
                  />
                )}
              />
            )}

            {/* Footer hint */}
            <View style={{ paddingHorizontal: 24, paddingVertical: 12, borderTopWidth: isIOS ? 0.33 : 1, borderTopColor: T.colors.separator }}>
              <Text style={{ fontSize: 11, color: T.colors.tertiaryLabel, textAlign: 'center', letterSpacing: 0.2 }}>
                {isIOS ? 'Tap a truck to reassign this order' : 'TAP A TRUCK TO REASSIGN THIS ORDER'}
              </Text>
            </View>
          </View>
        </View>
      </Modal>

      {/* ── Notification Panel Modal ─────────────────────────────────────── */}
      <NotificationsSheet
        visible={showNotifPanel}
        onClose={() => setShowNotifPanel(false)}
        theme={T}
        notifications={notifications}
        unreadCount={unreadCount}
        onMarkAllRead={markAllNotifsRead}
        onMarkRead={markNotifRead}
      />

      {/* ── Manifest Exceptions Panel Modal ──────────────────────────────── */}
      <ExceptionsSheet
        visible={showExceptionsPanel}
        onClose={() => setShowExceptionsPanel(false)}
        onRefresh={() => void loadManifestExceptions()}
        loading={loadingExceptions}
        exceptions={manifestExceptions}
        theme={T}
      />

      {toast}
      </View>
    </View>
  );
}
