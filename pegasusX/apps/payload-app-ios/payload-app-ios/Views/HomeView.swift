//
//  HomeView.swift
//  payload-app-ios
//
//  Phase 3 master-detail home for the iPad payload terminal. Sidebar lists
//  the payloader's assigned trucks; detail pane shows the selected truck's
//  open DRAFT/LOADING manifest workflow.
//

import SwiftUI

struct HomeView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.scenePhase) private var scenePhase
    @State private var viewModel = HomeViewModel()
    @State private var isTruckSidebarExpanded = true
    @State private var columnVisibility: NavigationSplitViewVisibility = .all

    var body: some View {
        VStack(spacing: 0) {
            PulseStrip(events: viewModel.pulseEvents, loading: viewModel.pulseLoading, error: viewModel.pulseError)
            navigationRoot
        }
        .overlay {
            if viewModel.startingLoading {
                Color.black.opacity(0.4).ignoresSafeArea()
                PayloadLoadingView(title: "STARTING LOADING", message: "Opening manifest for loading.")
                    .background(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusMD, style: .continuous))
                    .padding(32)
            } else if viewModel.sealingManifest {
                Color.black.opacity(0.4).ignoresSafeArea()
                PayloadLoadingView(title: "SEALING MANIFEST", message: "Finalizing load process.")
                    .background(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusMD, style: .continuous))
                    .padding(32)
            }
        }
        .homeSheets(viewModel: viewModel)
        .homeLifecycle(
            viewModel: viewModel,
            tokenStore: tokenStore,
            scenePhase: scenePhase
        )
    }

    private var navigationRoot: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            TruckSidebar(viewModel: viewModel, isExpanded: $isTruckSidebarExpanded)
                .navigationSplitViewColumnWidth(
                    min: isTruckSidebarExpanded ? 280 : 88,
                    ideal: isTruckSidebarExpanded ? 340 : 88,
                    max: isTruckSidebarExpanded ? 420 : 88
                )
                .navigationTitle(isTruckSidebarExpanded ? "Vehicles" : "")
                .toolbar { sidebarToolbar }
                .navigationDestination(for: String.self) { _ in
                    ManifestDetailPane(
                        viewModel: viewModel,
                        onShowException: { viewModel.presentExceptionSheet(for: $0) },
                        onShowReDispatch: { id in Task { await viewModel.openReDispatch(orderId: id) } },
                        onScanProduct: { viewModel.showProductScanner = true }
                    )
                    .navigationTitle("warehouse_portal.manifests.text.manifest")
                    .toolbar { detailToolbar }
                }
        } detail: {
            ManifestDetailPane(
                viewModel: viewModel,
                onShowException: { viewModel.presentExceptionSheet(for: $0) },
                onShowReDispatch: { id in Task { await viewModel.openReDispatch(orderId: id) } },
                onScanProduct: { viewModel.showProductScanner = true }
            )
            .navigationTitle("warehouse_portal.manifests.text.manifest")
            .navigationSplitViewColumnWidth(min: 320, ideal: 720)
            .toolbar { detailToolbar }
        }
        .navigationSplitViewStyle(.balanced)
    }

    @ToolbarContentBuilder
    private var sidebarToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarLeading) {
            OnlineDot(online: viewModel.online, queued: viewModel.queuedActions)
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                viewModel.toggleExceptionsPanel()
            } label: {
                Image(systemName: "exclamationmark.triangle")
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Manifest exceptions")
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                viewModel.toggleNotificationsPanel()
            } label: {
                Image(systemName: "bell")
                    .frame(minWidth: 44, minHeight: 44)
                    .overlay(alignment: .topTrailing) {
                        if viewModel.unreadCount > 0 {
                            Text("\(viewModel.unreadCount)")
                                .font(.caption2).bold()
                                .padding(.horizontal, 4)
                                .padding(.vertical, 1)
                                .background(.red)
                                .foregroundStyle(.white)
                                .clipShape(Capsule())
                                .offset(x: 6, y: -4)
                        }
                    }
            }
            .accessibilityLabel(viewModel.unreadCount > 0 ? "Notifications, \(viewModel.unreadCount) unread" : "Notifications")
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                PegasusHaptics.selection()
                viewModel.showInboundReturns = true
            } label: {
                Image(systemName: "arrow.uturn.backward.circle")
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Inbound returns")
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                PegasusHaptics.light()
                Task { await viewModel.refreshTrucks() }
            } label: {
                Image(systemName: "arrow.clockwise")
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Refresh vehicles")
        }
    }

    @ToolbarContentBuilder
    private var detailToolbar: some ToolbarContent {
        if viewModel.manifest?.state == "LOADING" {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.selection()
                    viewModel.showInjectSheet = true
                } label: {
                    Image(systemName: "plus.circle")
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel("Inject order")
            }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                Button("mobile_payload.ui.refresh_manifest") {
                    PegasusHaptics.light()
                    Task { await viewModel.refreshManifest() }
                }
                Button("mobile_payload.ui.logout", role: .destructive) {
                    PegasusHaptics.warning()
                    tokenStore.logout()
                }
            } label: {
                Image(systemName: "person.crop.circle")
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel("Operator account menu")
        }
    }
}

private struct HomeSheetsModifier: ViewModifier {
    @Bindable var viewModel: HomeViewModel

    func body(content: Content) -> some View {
        content
            .sheet(isPresented: $viewModel.showInjectSheet) {
                InjectOrderSheet(
                    injecting: viewModel.injectingOrder,
                    onCancel: { viewModel.showInjectSheet = false },
                    onSubmit: { id in
                        Task {
                            await viewModel.injectOrder(id)
                            viewModel.showInjectSheet = false
                        }
                    }
                )
            }
            .sheet(isPresented: $viewModel.showProductScanner) {
                ProductScannerSheet(viewModel: viewModel, isPresented: $viewModel.showProductScanner)
            }
            .sheet(isPresented: $viewModel.showInboundReturns) {
                InboundReturnsView(online: viewModel.online)
            }
            .sheet(item: $viewModel.exceptionTarget) { target in
                ExceptionReasonSheet(
                    orderId: target.id,
                    inFlight: viewModel.exceptionLoadingOrderId == target.id,
                    onCancel: { viewModel.dismissExceptionSheet() },
                    onSelect: { reason in
                        Task {
                            await viewModel.reportException(orderId: target.id, reason: reason)
                            viewModel.dismissExceptionSheet()
                        }
                    }
                )
            }
            .sheet(item: $viewModel.reDispatchTarget) { target in
                ReDispatchSheet(
                    orderId: target.id,
                    loading: viewModel.loadingRecommendations,
                    response: viewModel.recommendations,
                    reassigning: viewModel.reassigning,
                    onClose: { viewModel.closeReDispatch() },
                    onPick: { driverId, isPartial in
                        Task { await viewModel.reassignTo(driverId, isPartial: isPartial) }
                    }
                )
            }
            .onChange(of: viewModel.reDispatchTarget) { _, target in
                if target == nil && viewModel.reDispatchOrderId != nil {
                    viewModel.closeReDispatch()
                }
            }
            .alert(
                "DLQ Escalation",
                isPresented: $viewModel.isShowingEscalatedAlert,
                actions: { Button("OK", role: .cancel) { viewModel.clearEscalatedMessage() } },
                message: { Text(viewModel.escalatedMessage ?? "") }
            )
            .alert(
                "Handoff",
                isPresented: $viewModel.isShowingHandoffAlert,
                actions: { Button("OK", role: .cancel) { viewModel.clearHandoffNavigationMessage() } },
                message: { Text(viewModel.handoffNavigationMessage ?? "") }
            )
            .sheet(isPresented: $viewModel.showNotificationsPanel) {
                NotificationsSheet(viewModel: viewModel)
            }
            .sheet(isPresented: $viewModel.showExceptionsPanel) {
                ManifestExceptionsSheet(viewModel: viewModel)
            }
    }
}

private struct HomeLifecycleModifier: ViewModifier {
    @Bindable var viewModel: HomeViewModel
    let tokenStore: TokenStore
    let scenePhase: ScenePhase

    func body(content: Content) -> some View {
        content
            .overlay(alignment: .bottom) { HomeBannerOverlay(viewModel: viewModel) }
            .task {
                await viewModel.refreshTrucks()
                if let token = tokenStore.token {
                    await viewModel.bootstrapPhase6(token: token)
                }
                let targetVM = viewModel
                PushNotificationManager.shared.onOpenPanel = { [weak targetVM] in
                    guard let targetVM else { return }
                    if !targetVM.showNotificationsPanel { targetVM.toggleNotificationsPanel() }
                }
                await PushNotificationManager.shared.requestAuthorization()
            }
            .onChange(of: scenePhase) { _, phase in
                guard phase == .active else { return }
                Task {
                    await viewModel.refreshTrucks(silent: !viewModel.trucks.isEmpty)
                    if viewModel.selectedTruckId != nil {
                        await viewModel.refreshManifest(silent: viewModel.manifest != nil || !viewModel.orders.isEmpty)
                    }
                }
            }
            .onChange(of: viewModel.online) { _, online in
                guard online else { return }
                Task {
                    await viewModel.refreshTrucks(silent: !viewModel.trucks.isEmpty)
                    if viewModel.selectedTruckId != nil {
                        await viewModel.refreshManifest(silent: viewModel.manifest != nil || !viewModel.orders.isEmpty)
                    }
                }
            }
            .onDisappear { viewModel.disconnectPhase6() }
    }
}

private extension View {
    func homeSheets(viewModel: HomeViewModel) -> some View {
        modifier(HomeSheetsModifier(viewModel: viewModel))
    }

    func homeLifecycle(
        viewModel: HomeViewModel,
        tokenStore: TokenStore,
        scenePhase: ScenePhase
    ) -> some View {
        modifier(HomeLifecycleModifier(
            viewModel: viewModel,
            tokenStore: tokenStore,
            scenePhase: scenePhase
        ))
    }
}
