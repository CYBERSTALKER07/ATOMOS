//
//  HomeView.swift
//  payload-app-ios
//
//  Phase 3 master-detail home for the iPad payload terminal. Sidebar lists
//  the payloader's assigned trucks; detail pane shows the selected truck's
//  open DRAFT manifest summary. Loading workflow lands in Phase 4.
//

import SwiftUI

struct HomeView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.scenePhase) private var scenePhase
    @State private var viewModel = HomeViewModel()
    @State private var isTruckSidebarExpanded = true
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
    @State private var showInjectSheet = false
    @State private var showProductScanner = false
    @State private var showInboundReturns = false
    @State private var exceptionTargetOrderId: String?

    var body: some View {
        VStack(spacing: 0) {
            PulseStrip(events: viewModel.pulseEvents, loading: viewModel.pulseLoading)
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
            .homeSheets(
                viewModel: viewModel,
                tokenStore: tokenStore,
                showInjectSheet: $showInjectSheet,
                showProductScanner: $showProductScanner,
                showInboundReturns: $showInboundReturns,
                exceptionTargetOrderId: $exceptionTargetOrderId
            )
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
        } detail: {
            ManifestDetailPane(
                viewModel: viewModel,
                onShowException: { exceptionTargetOrderId = $0 },
                onShowReDispatch: { id in Task { await viewModel.openReDispatch(orderId: id) } },
                onScanProduct: { showProductScanner = true }
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
            Button { viewModel.toggleExceptionsPanel() } label: {
                Image(systemName: "exclamationmark.triangle")
            }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button { viewModel.toggleNotificationsPanel() } label: {
                Image(systemName: "bell")
                    .overlay(alignment: .topTrailing) {
                        if viewModel.unreadCount > 0 {
                            Text("\(viewModel.unreadCount)")
                                .font(.caption2).bold()
                                .padding(.horizontal, 4).padding(.vertical, 1)
                                .background(.red).foregroundStyle(.white)
                                .clipShape(Capsule())
                                .offset(x: 10, y: -8)
                        }
                    }
            }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button { showInboundReturns = true } label: {
                Image(systemName: "arrow.uturn.backward.circle")
            }
            .accessibilityLabel("Inbound returns")
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button { Task { await viewModel.refreshTrucks() } } label: {
                Image(systemName: "arrow.clockwise")
            }
        }
    }

    @ToolbarContentBuilder
    private var detailToolbar: some ToolbarContent {
        if viewModel.manifest?.state == "LOADING" {
            ToolbarItem(placement: .topBarTrailing) {
                Button { showInjectSheet = true } label: {
                    Image(systemName: "plus.circle")
                }
                .accessibilityLabel("Inject order")
            }
        }
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                Button("mobile_payload.ui.refresh_manifest") {
                    Task { await viewModel.refreshManifest() }
                }
                Button("mobile_payload.ui.logout", role: .destructive) {
                    tokenStore.logout()
                }
            } label: {
                Image(systemName: "person.crop.circle")
            }
        }
    }
}

private struct HomeSheetsModifier: ViewModifier {
    @Bindable var viewModel: HomeViewModel
    let tokenStore: TokenStore
    @Binding var showInjectSheet: Bool
    @Binding var showProductScanner: Bool
    @Binding var showInboundReturns: Bool
    @Binding var exceptionTargetOrderId: String?

    func body(content: Content) -> some View {
        content
            .sheet(isPresented: $showInjectSheet) {
                InjectOrderSheet(
                    injecting: viewModel.injectingOrder,
                    onCancel: { showInjectSheet = false },
                    onSubmit: { id in
                        Task {
                            await viewModel.injectOrder(id)
                            showInjectSheet = false
                        }
                    }
                )
            }
            .sheet(isPresented: $showProductScanner) {
                ProductScannerSheet(viewModel: viewModel, isPresented: $showProductScanner)
            }
            .sheet(isPresented: $showInboundReturns) {
                InboundReturnsView(online: viewModel.online)
            }
            .sheet(item: Binding(
                get: { exceptionTargetOrderId.map { ExceptionTarget(id: $0) } },
                set: { exceptionTargetOrderId = $0?.id }
            )) { target in
                ExceptionReasonSheet(
                    orderId: target.id,
                    inFlight: viewModel.exceptionLoadingOrderId == target.id,
                    onCancel: { exceptionTargetOrderId = nil },
                    onSelect: { reason in
                        Task {
                            await viewModel.reportException(orderId: target.id, reason: reason)
                            exceptionTargetOrderId = nil
                        }
                    }
                )
            }
            .sheet(item: Binding(
                get: { viewModel.reDispatchOrderId.map { ReDispatchTarget(id: $0) } },
                set: { if $0 == nil { viewModel.closeReDispatch() } }
            )) { target in
                ReDispatchSheet(
                    orderId: target.id,
                    loading: viewModel.loadingRecommendations,
                    response: viewModel.recommendations,
                    reassigning: viewModel.reassigning,
                    onClose: { viewModel.closeReDispatch() },
                    onPick: { driverId, isPartial in Task { await viewModel.reassignTo(driverId, isPartial: isPartial) } }
                )
            }
            .alert(
                "DLQ Escalation",
                isPresented: Binding(
                    get: { viewModel.escalatedMessage != nil },
                    set: { if !$0 { viewModel.clearEscalatedMessage() } }
                ),
                actions: { Button("OK", role: .cancel) { viewModel.clearEscalatedMessage() } },
                message: { Text(viewModel.escalatedMessage ?? "") }
            )
            .alert(
                "Handoff",
                isPresented: Binding(
                    get: { viewModel.handoffNavigationMessage != nil },
                    set: { if !$0 { viewModel.clearHandoffNavigationMessage() } }
                ),
                actions: { Button("OK", role: .cancel) { viewModel.clearHandoffNavigationMessage() } },
                message: { Text(viewModel.handoffNavigationMessage ?? "") }
            )
            .sheet(isPresented: Binding(
                get: { viewModel.showNotificationsPanel },
                set: { viewModel.showNotificationsPanel = $0 }
            )) {
                NotificationsSheet(viewModel: viewModel)
            }
            .sheet(isPresented: Binding(
                get: { viewModel.showExceptionsPanel },
                set: { viewModel.showExceptionsPanel = $0 }
            )) {
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
                PushNotificationManager.shared.onOpenPanel = { [weak viewModel] in
                    guard let viewModel else { return }
                    if !viewModel.showNotificationsPanel { viewModel.toggleNotificationsPanel() }
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

private struct HomeBannerOverlay: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        if viewModel.barcodeScanMessage != nil ||
           viewModel.missingItemsReportedMessage != nil ||
           viewModel.queuedNoticeMessage != nil ||
           viewModel.syncCompleteMessage != nil {
            VStack(spacing: 8) {
                if let msg = viewModel.barcodeScanMessage {
                    InfoBanner(text: msg, tint: .blue)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .task {
                            try? await Task.sleep(nanoseconds: 3_000_000_000)
                            viewModel.clearBarcodeScanMessage()
                        }
                }
                if let msg = viewModel.missingItemsReportedMessage {
                    InfoBanner(text: msg, tint: .orange)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .task {
                            try? await Task.sleep(nanoseconds: 3_000_000_000)
                            viewModel.clearMissingItemsReportedMessage()
                        }
                }
                if let msg = viewModel.queuedNoticeMessage {
                    InfoBanner(text: msg, tint: .orange)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .task {
                            try? await Task.sleep(nanoseconds: 3_000_000_000)
                            viewModel.clearQueuedNoticeMessage()
                        }
                }
                if let msg = viewModel.syncCompleteMessage {
                    InfoBanner(text: msg, tint: .green)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .task {
                            try? await Task.sleep(nanoseconds: 3_000_000_000)
                            viewModel.clearSyncCompleteMessage()
                        }
                }
            }
            .animation(.easeInOut, value: viewModel.queuedNoticeMessage)
            .animation(.easeInOut, value: viewModel.syncCompleteMessage)
            .animation(.easeInOut, value: viewModel.barcodeScanMessage)
            .padding()
        }
    }
}

private struct ProductScannerSheet: View {
    @Bindable var viewModel: HomeViewModel
    @Binding var isPresented: Bool

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Text("mobile_payload.ui.scan_product_ean")
                    .font(.system(size: 14, weight: .black, design: .monospaced))
                EANBarcodeScannerView(onBarcode: { code in
                    Task {
                        await viewModel.onBarcodeScanned(code)
                        isPresented = false
                    }
                })
                Spacer()
            }
            .padding()
            .navigationTitle("mobile_payload.ui.product_scan")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { isPresented = false }
                }
            }
        }
    }
}

private extension View {
    func homeSheets(
        viewModel: HomeViewModel,
        tokenStore: TokenStore,
        showInjectSheet: Binding<Bool>,
        showProductScanner: Binding<Bool>,
        showInboundReturns: Binding<Bool>,
        exceptionTargetOrderId: Binding<String?>
    ) -> some View {
        modifier(HomeSheetsModifier(
            viewModel: viewModel,
            tokenStore: tokenStore,
            showInjectSheet: showInjectSheet,
            showProductScanner: showProductScanner,
            showInboundReturns: showInboundReturns,
            exceptionTargetOrderId: exceptionTargetOrderId
        ))
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

private struct ExceptionTarget: Identifiable { let id: String }
private struct ReDispatchTarget: Identifiable { let id: String }

// ── Sidebar ──────────────────────────────────────────────────────────────

// MARK: - Detail

/*
private struct ManifestDetailView: View {
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void
    let onScanProduct: () -> Void

    var body: some View {
        Group {
            if viewModel.selectedTruckId == nil {
                PayloadStateView(
                    variant: .truck,
                    title: "SELECT_A_VEHICLE",
                    message: "Pick a truck from the sidebar to load its manifest.",
                    compact: false
                )
            } else if viewModel.manifestSealed {
                AllSealedSuccessView(
                    dispatchCodes: viewModel.dispatchCodes,
                    onStartNew: { Task { await viewModel.startNewManifest() } }
                )
            } else if viewModel.loadingManifest && viewModel.manifest == nil {
                PayloadLoadingView(
                    title: "LOADING_MANIFEST",
                    message: "Loading the active checklist for this truck."
                )
            } else if let m = viewModel.manifest {
                ManifestWorkflow(
                    manifest: m,
                    truck: viewModel.trucks.first { $0.id == viewModel.selectedTruckId },
                    viewModel: viewModel,
                    onShowException: onShowException,
                    onShowReDispatch: onShowReDispatch,
                    onScanProduct: onScanProduct
                )
            } else {
                PayloadStateView(
                    variant: .manifest,
                    title: "NO_OPEN_MANIFEST",
                    message: "This vehicle has no DRAFT or LOADING manifest. Wait for dispatch.",
                    tone: .warning
                )
            }
        }
    }
}

private struct ManifestWorkflow: View {
    let manifest: Manifest
    let truck: Truck?
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void
    let onScanProduct: () -> Void

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) { // Increased tactical spacing
                if let truck { TruckHeader(truck: truck) }

                ManifestKpiGrid(manifest: manifest)

                if viewModel.error != nil || viewModel.errorExplain != nil {
                    ExplainStatusBanner(
                        explain: viewModel.errorExplain,
                        fallbackTitle: viewModel.error,
                        fallbackDetail: nil
                    )
                }

                if manifest.state == "DRAFT" {
                    Button {
                        Task { await viewModel.startLoading() }
                    } label: {
                        Text("mobile_payload.ui.start_loading")
                            .font(.headline)
                            .frame(maxWidth: .infinity, minHeight: 48)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.startingLoading)
                    Text("mobile_payload.ui.tap_start_loading_to_open_the_manifest_for_tap_check_and_per_ord")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                } else if manifest.state == "LOADING" || manifest.state == "SEALED" {
                    if let psId = viewModel.postSealOrderId {
                        PostSealCountdownView(
                            orderId: psId,
                            dispatchCode: viewModel.dispatchCodes[psId] ?? "",
                            secondsLeft: viewModel.postSealCountdown,
                            reportingMissingItems: viewModel.reportingMissingItems,
                            onDismiss: { viewModel.dismissCountdown() },
                            onReportMissingItems: {
                                Task { await viewModel.reportMissingItems(orderId: psId) }
                            }
                        )
                    }

                    OrderChecklistSection(
                        viewModel: viewModel,
                        onShowException: onShowException,
                        onShowReDispatch: onShowReDispatch,
                        onScanProduct: onScanProduct
                    )

                    if viewModel.allOrdersSealed && manifest.state != "SEALED" {
                        Button {
                            Task { await viewModel.sealManifest() }
                        } label: {
                            HStack {
                                Image(systemName: "lock.fill")
                                Text("mobile_payload.ui.seal_manifest").font(.headline)
                            }
                            .frame(maxWidth: .infinity, minHeight: 48)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.green)
                        .disabled(viewModel.sealingManifest)
                    }
                }
            }
            .padding()
        }
    }
}


// MARK: - Post-seal 60s countdown

private struct PostSealCountdownView: View {
    let orderId: String
    let dispatchCode: String
    let secondsLeft: Int
    let reportingMissingItems: Bool
    let onDismiss: () -> Void
    let onReportMissingItems: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            PayloadStateView(
                variant: .warning,
                title: "ORDER_SEALED",
                message: "Double-check the load before dispatch.",
                compact: true,
                tone: .warning
            )

            HStack {
                Image(systemName: "lock.shield.fill")
                    .foregroundStyle(TermTheme.live)
                Text("ORDER_SEALED")
                    .font(.system(size: 12, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
                Spacer()
                Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                    .font(.system(size: 12, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
            }
            
            VStack(alignment: .leading, spacing: 4) {
                Text("DISPATCH_CODE")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
                
                Text(dispatchCode)
                    .font(.system(size: 44, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.live)
            }
            
            HStack {
                Text(L10n.format("mobile_payload.ui.double_check_window_secondslefts_2", "\(secondsLeft)"))
                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
                Spacer()
            }
            
            ProgressView(value: Double(secondsLeft) / 60.0)
                .tint(TermTheme.live)

            Button {
                onReportMissingItems()
            } label: {
                HStack {
                    if reportingMissingItems {
                        ProgressView().controlSize(.small)
                    }
                    Text("REPORT_MISSING_ITEMS")
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(TermTheme.card)
                .foregroundStyle(TermTheme.accent)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            .disabled(reportingMissingItems)
            
            Button {
                onDismiss()
            } label: {
                Text("CONTINUE_TO_NEXT")
                    .font(.system(size: 14, weight: .black, design: .monospaced))
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(TermTheme.accent)
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
        }
        .padding(20)
        .background(TermTheme.live.opacity(0.05))
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .stroke(TermTheme.live.opacity(0.2), lineWidth: 1)
        }
    }
}

// MARK: - All Sealed success terminal screen

private struct AllSealedSuccessView: View {
    let dispatchCodes: [String: String]
    let onStartNew: () -> Void

    var body: some View {
        ScrollView {
            VStack(spacing: 32) {
                PayloadStateView(
                    variant: .success,
                    title: "MANIFEST_LOCKED",
                    message: "All items verified and sealed for transport.",
                    tone: .success
                )
                .padding(.top, 40)

                VStack(alignment: .leading, spacing: 16) {
                    Text("DISPATCH_MANIFEST_SUMMARY")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    
                    VStack(spacing: 8) {
                        ForEach(dispatchCodes.sorted(by: { $0.key < $1.key }), id: \.key) { id, code in
                            HStack {
                                Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(id.suffix(6).uppercased())"))
                                    .font(.system(size: 14, weight: .black, design: .monospaced))
                                    .foregroundStyle(TermTheme.accent)
                                Spacer()
                                Text(code)
                                    .font(.system(size: 18, weight: .black, design: .monospaced))
                                    .foregroundStyle(TermTheme.live)
                            }
                            .padding(16)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: 12, style: .continuous)
                                    .stroke(TermTheme.separator.opacity(0.1), lineWidth: 1)
                            }
                        }
                    }
                }
                .padding(24)
                .background(TermTheme.bg)
                .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
                .overlay {
                    RoundedRectangle(cornerRadius: 24, style: .continuous)
                        .stroke(TermTheme.separator.opacity(0.1), lineWidth: 1)
                }

                Button {
                    onStartNew()
                } label: {
                    HStack {
                        Image(systemName: "plus.circle.fill")
                        Text("START_NEXT_LOAD")
                            .font(.system(size: 18, weight: .black, design: .monospaced))
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 20)
                    .background(TermTheme.accent)
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                .buttonStyle(.tactical)
            }
            .padding(24)
        }
        .background(TermTheme.bg)
    }
}

private struct TruckHeader: View {
    let truck: Truck
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(truck.label?.uppercased() ?? truck.licensePlate?.uppercased() ?? "VEHICLE-\(truck.id.prefix(8).uppercased())")
                .font(.system(size: 28, weight: .black, design: .monospaced)) // Massive tactical header
                .foregroundStyle(TermTheme.accent)
                .tracking(1.4)
            
            HStack(spacing: 8) {
                if let p = truck.licensePlate, !p.isEmpty {
                    Text(p.uppercased())
                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                if let c = truck.vehicleClass, !c.isEmpty {
                    Text("—")
                        .foregroundStyle(TermTheme.tertiary)
                    Text(c.uppercased())
                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                Spacer()
                
                // Active Marker
                HStack(spacing: 6) {
                    Circle()
                        .fill(TermTheme.live)
                        .frame(width: 8, height: 8)
                    Text("ACTIVE_NODE")
                        .font(.system(size: 10, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.live)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .background(TermTheme.live.opacity(0.12), in: Capsule())
            }
        }
        .padding(TermTheme.s24)
        .tacticalCard(radius: TermTheme.radiusMD)
    }
}
*/

// MARK: - Phase 5 sheets

private struct InjectOrderSheet: View {
    let injecting: Bool
    let onCancel: () -> Void
    let onSubmit: (String) -> Void
    @State private var orderId: String = ""
    @State private var showScanner = false

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()
            
            VStack(spacing: 24) {
                // Tactical Header
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("SYSTEM_INJECTION")
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                        Text("INJECT_ORDER")
                            .font(.system(size: 24, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.accent)
                    }
                    Spacer()
                    Button(action: onCancel) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 28))
                            .foregroundStyle(TermTheme.tertiary)
                    }
                    .disabled(injecting)
                }
                .padding(.horizontal, 4)

                VStack(alignment: .leading, spacing: 16) {
                    Text("TARGET_ORDER_ID")
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                    
                    TextField("mobile_payload.ui.ord_xxxxxx", text: $orderId)
                        .font(.system(size: 20, weight: .black, design: .monospaced))
                        .padding(16)
                        .background(TermTheme.card)
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .overlay {
                            RoundedRectangle(cornerRadius: 12, style: .continuous)
                                .stroke(TermTheme.accent.opacity(0.1), lineWidth: 1)
                        }
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .disabled(injecting)
                    
                    Text("mobile_payload.ui.add_an_order_mid_load_scan_an_order_label_or_enter_the_order_id")
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                        .padding(.horizontal, 4)

                    Button {
                        showScanner.toggle()
                    } label: {
                        Label(showScanner ? "HIDE SCANNER" : "SCAN ORDER LABEL", systemImage: "barcode.viewfinder")
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(injecting)

                    if showScanner {
                        EANBarcodeScannerView(onBarcode: { scanned in
                            orderId = scanned.trimmingCharacters(in: .whitespacesAndNewlines)
                            showScanner = false
                        })
                    }
                }
                .padding(20)
                .background(TermTheme.card.opacity(0.5))
                .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
                .tacticalCard()

                Spacer()

                Button {
                    onSubmit(orderId)
                } label: {
                    HStack {
                        if injecting {
                            ProgressView().tint(TermTheme.card)
                        } else {
                            Image(systemName: "bolt.fill")
                            Text("EXECUTE_INJECTION")
                                .font(.system(size: 16, weight: .black, design: .monospaced))
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 18)
                    .background(orderId.trimmingCharacters(in: .whitespaces).isEmpty ? TermTheme.tertiary : TermTheme.accent)
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                .disabled(injecting || orderId.trimmingCharacters(in: .whitespaces).isEmpty)
                .buttonStyle(.tactical)
            }
            .padding(24)
        }
    }
}

private struct ExceptionReasonSheet: View {
    let orderId: String
    let inFlight: Bool
    let onCancel: () -> Void
    let onSelect: (String) -> Void

    private let reasons: [(code: String, label: String, icon: String)] = [
        ("OVERFLOW", "OVERFLOW - NO CAPACITY", "shippingbox.fill"),
        ("DAMAGED", "DAMAGED - QUALITY FAIL", "exclamationmark.shield.fill"),
        ("MANUAL", "MANUAL - OPERATOR VOID", "hand.raised.fill"),
    ]

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()
            
            VStack(spacing: 24) {
                // Header
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("EXCEPTION_REPORT")
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                        Text(L10n.format("mobile_payload.ui.remove_ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                            .font(.system(size: 20, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.warn)
                    }
                    Spacer()
                    Button(action: onCancel) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 28))
                            .foregroundStyle(TermTheme.tertiary)
                    }
                }
                .padding(.horizontal, 4)

                VStack(alignment: .leading, spacing: 16) {
                    Text("SELECT_EXCEPTION_REASON")
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                    
                    VStack(spacing: 12) {
                        ForEach(reasons, id: \.code) { reason in
                            Button {
                                onSelect(reason.code)
                            } label: {
                                HStack(spacing: 16) {
                                    Image(systemName: reason.icon)
                                        .font(.system(size: 20))
                                        .foregroundStyle(TermTheme.warn)
                                        .frame(width: 44, height: 44)
                                        .background(TermTheme.warn.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                                    
                                    Text(reason.label)
                                        .font(.system(size: 14, weight: .black, design: .monospaced))
                                        .foregroundStyle(TermTheme.accent)
                                    
                                    Spacer()
                                    
                                    if inFlight {
                                        ProgressView().tint(TermTheme.accent)
                                    } else {
                                        Image(systemName: "chevron.right")
                                            .font(.system(size: 14, weight: .bold))
                                            .foregroundStyle(TermTheme.tertiary)
                                    }
                                }
                                .padding(12)
                                .background(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                                .tacticalCard()
                            }
                            .disabled(inFlight)
                            .buttonStyle(.tactical)
                        }
                    }
                    
                    Text("mobile_payload.ui.3_overflow_attempts_on_this_manifest_will_escalate_to_admin_dlq")
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                        .padding(.horizontal, 4)
                        .padding(.top, 8)
                }
                
                Spacer()
            }
            .padding(24)
        }
    }
}

private struct ReDispatchSheet: View {
    let orderId: String
    let loading: Bool
    let response: RecommendReassignResponse?
    let reassigning: Bool
    let onClose: () -> Void
    let onPick: (String, Bool) -> Void

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()
            
            VStack(spacing: 24) {
                // Header
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("LOGISTICS_OPTIMIZER")
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                        Text(L10n.format("mobile_payload.ui.re_dispatch_ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                            .font(.system(size: 20, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.accent)
                    }
                    Spacer()
                    Button(action: onClose) {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 28))
                            .foregroundStyle(TermTheme.tertiary)
                    }
                }
                .padding(.horizontal, 4)

                Group {
                    if loading {
                        PayloadStateView(
                            variant: .dispatch,
                            title: "SOLVING_CONSTRAINTS...",
                            message: "Scoring nearby trucks for the best reassignment path.",
                            compact: true
                        )
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else if let resp = response {
                        ScrollView {
                            VStack(spacing: 20) {
                                // Order Info Card
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("TARGET_OBJECT")
                                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                                        .foregroundStyle(TermTheme.secondary)
                                    
                                    HStack {
                                        Text(resp.retailerName?.uppercased() ?? "OFFLINE_RETAILER")
                                            .font(.system(size: 16, weight: .black, design: .monospaced))
                                            .foregroundStyle(TermTheme.accent)
                                        Spacer()
                                        Text(String(format: "%.1f VU", resp.orderVolumeVu ?? 0))
                                            .font(.system(size: 16, weight: .black, design: .monospaced))
                                            .foregroundStyle(TermTheme.accent)
                                    }
                                }
                                .padding(16)
                                .background(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                                .tacticalCard()

                                if (resp.recommendations).isEmpty {
                                    PayloadStateView(
                                        variant: .dispatch,
                                        title: "NO_SUITABLE_CARRIERS_FOUND",
                                        message: "No nearby fleet target can accept this order right now.",
                                        compact: true,
                                        tone: .warning
                                    )
                                    .padding(40)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.card)
                                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                                    .tacticalCard()
                                } else {
                                    VStack(alignment: .leading, spacing: 12) {
                                        Text("AI_RECOMMENDATIONS")
                                            .font(.system(size: 12, weight: .black, design: .monospaced))
                                            .foregroundStyle(TermTheme.secondary)
                                            .padding(.horizontal, 4)
                                        
                                        VStack(spacing: 12) {
                                            ForEach(resp.recommendations) { rec in
                                                RecommendationRow(
                                                    rec: rec,
                                                    onPickComplete: { onPick(rec.driverId, false) },
                                                    onPickPartial: { onPick(rec.driverId, true) }
                                                )
                                                .disabled(reassigning)
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    } else {
                        PayloadStateView(
                            variant: .dispatch,
                            title: "NO_DATA_AVAILABLE",
                            message: "The optimizer response is not available yet.",
                            compact: true,
                            tone: .warning
                        )
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    }
                }
            }
            .padding(24)
            
            if reassigning {
                TermTheme.bg.opacity(0.8).ignoresSafeArea()
                VStack(spacing: 16) {
                    ProgressView().tint(TermTheme.accent)
                    Text("mobile_payload.ui.reassigning_order")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                }
            }
        }
    }
}



// MARK: - Phase 6: notifications sheet / info banner

private struct InfoBanner: View {
    let text: String
    let tint: Color
    var body: some View {
        Text(text)
            .font(.footnote.bold())
            .foregroundStyle(.white)
            .padding(.horizontal, 16).padding(.vertical, 10)
            .background(tint)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .shadow(radius: 4)
    }
}
