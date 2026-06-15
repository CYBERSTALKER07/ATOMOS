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
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
    @State private var showInjectSheet = false
    @State private var exceptionTargetOrderId: String?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(message: viewModel.clientPolicyMessage)
            NavigationSplitView(columnVisibility: $columnVisibility) {
            TruckSidebar(viewModel: viewModel)
                .navigationSplitViewColumnWidth(min: 280, ideal: 340, max: 420)
                .navigationTitle("Vehicles")
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        OnlineDot(online: viewModel.online, queued: viewModel.queuedActions)
                    }
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            viewModel.toggleExceptionsPanel()
                        } label: {
                            Image(systemName: "exclamationmark.triangle")
                        }
                    }
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            viewModel.toggleNotificationsPanel()
                        } label: {
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
                        Button {
                            Task { await viewModel.refreshTrucks() }
                        } label: {
                            Image(systemName: "arrow.clockwise")
                        }
                    }
                }
        } detail: {
            ManifestDetailView(
                viewModel: viewModel,
                onShowException: { exceptionTargetOrderId = $0 },
                onShowReDispatch: { id in Task { await viewModel.openReDispatch(orderId: id) } }
            )
            .navigationTitle("Manifest")
            .navigationSplitViewColumnWidth(min: 480, ideal: 720)
            .toolbar {
                if viewModel.manifest?.state == "LOADING" {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            showInjectSheet = true
                        } label: {
                            Image(systemName: "plus.circle")
                        }
                        .accessibilityLabel("Inject order")
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button("Refresh manifest") {
                            Task { await viewModel.refreshManifest() }
                        }
                        Button("Logout", role: .destructive) {
                            tokenStore.logout()
                        }
                    } label: {
                        Image(systemName: "person.crop.circle")
                    }
                }
            }
        }
        .navigationSplitViewStyle(.balanced)
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
                onPick: { driverId in Task { await viewModel.reassignTo(driverId) } }
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
        .overlay(alignment: .bottom) {
            VStack(spacing: 8) {
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
            .padding()
        }
        .task {
            await viewModel.refreshTrucks()
            if let token = tokenStore.token {
                await viewModel.bootstrapPhase6(token: token)
            }
            // Phase 7: request APNs authorization and route tap-actions into the panel.
            PushNotificationManager.shared.onOpenPanel = { [weak viewModel] in
                guard let viewModel else { return }
                if !viewModel.showNotificationsPanel { viewModel.toggleNotificationsPanel() }
            }
            await PushNotificationManager.shared.requestAuthorization()
        }
        .onChange(of: scenePhase) { _, phase in
            // Re-fetch on foreground resume so stale manifests are never shown.
            if phase == .active {
                Task {
                    await viewModel.refreshTrucks()
                    if viewModel.selectedTruckId != nil {
                        await viewModel.refreshManifest()
                    }
                }
            }
        }
        .onChange(of: viewModel.online) { _, online in
            // Re-fetch after every WS reconnect to drain any missed mutations.
            if online {
                Task {
                    await viewModel.refreshTrucks()
                    if viewModel.selectedTruckId != nil {
                        await viewModel.refreshManifest()
                    }
                }
            }
        }
        .onDisappear { viewModel.disconnectPhase6() }
        }
    }
}

private struct ExceptionTarget: Identifiable { let id: String }
private struct ReDispatchTarget: Identifiable { let id: String }

// MARK: - Sidebar

private struct TruckSidebar: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        Group {
            if viewModel.loadingTrucks && viewModel.trucks.isEmpty {
                PayloadStateView(
                    variant: .truck,
                    title: "LOADING_VEHICLES",
                    message: "Refreshing supplier fleet availability for this shift."
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.trucks.isEmpty {
                PayloadStateView(
                    variant: .truck,
                    title: "NO_VEHICLES",
                    message: "Pull to refresh once dispatch assigns trucks.",
                    tone: .warning
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                if viewModel.batchReadyManifestIds.count > 1 {
                    Section {
                        VStack(alignment: .leading, spacing: TermTheme.s8) {
                            PayloadSectionHeader(
                                title: "BATCH FINALIZE",
                                subtitle: "\(viewModel.batchReadyManifestIds.count) trucks ready to seal"
                            )
                            Button(viewModel.batchSealing ? "Finalizing…" : "Seal all trucks") {
                                Task { await viewModel.finalizeBatchSeal() }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(viewModel.batchSealing)
                        }
                        .padding(.vertical, 4)
                    }
                }
                List(viewModel.trucks, selection: Binding(
                    get: { viewModel.selectedTruckId },
                    set: { id in if let id { Task { await viewModel.selectTruck(id) } } }
                )) { truck in
                    TruckRow(truck: truck)
                        .tag(truck.id)
                }
                .refreshable { await viewModel.refreshTrucks() }
            }
            if let err = viewModel.error {
                Text(err)
                    .font(.footnote)
                    .foregroundStyle(.red)
                    .padding(.horizontal)
            }
        }
    }
}

private struct TruckRow: View {
    let truck: Truck

    var body: some View {
        HStack(spacing: 16) {
            ZStack {
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(TermTheme.accent.opacity(0.06))
                    .frame(width: 48, height: 48)
                
                Image(systemName: "truck.box.fill")
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(TermTheme.accent)
            }

            VStack(alignment: .leading, spacing: 4) {
                Text(displayLabel.uppercased())
                    .font(.system(size: 16, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
                
                Text(subtitle.uppercased())
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundStyle(TermTheme.secondary)
            }
        }
        .padding(.vertical, 8)
    }

    private var displayLabel: String {
        if let l = truck.label, !l.isEmpty { return l }
        if let p = truck.licensePlate, !p.isEmpty { return p }
        return "TRK-\(truck.id.prefix(6))"
    }

    private var subtitle: String {
        [truck.licensePlate, truck.vehicleClass]
            .compactMap { $0?.isEmpty == false ? $0 : nil }
            .joined(separator: " — ")
    }
}

// MARK: - Detail

private struct ManifestDetailView: View {
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void

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
                PayloadStateView(
                    variant: .manifest,
                    title: "LOADING_MANIFEST",
                    message: "Loading the active checklist for this truck."
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let m = viewModel.manifest {
                ManifestWorkflow(
                    manifest: m,
                    truck: viewModel.trucks.first { $0.id == viewModel.selectedTruckId },
                    viewModel: viewModel,
                    onShowException: onShowException,
                    onShowReDispatch: onShowReDispatch
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

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) { // Increased tactical spacing
                if let truck { TruckHeader(truck: truck) }

                ManifestKpiGrid(manifest: manifest)

                if let err = viewModel.error {
                    Text(err)
                        .font(.footnote)
                        .foregroundStyle(.red)
                        .padding(.horizontal, 8)
                }

                if manifest.state == "DRAFT" {
                    Button {
                        Task { await viewModel.startLoading() }
                    } label: {
                        if viewModel.startingLoading {
                            ProgressView().controlSize(.regular)
                        } else {
                            Text("Start Loading")
                                .font(.headline)
                                .frame(maxWidth: .infinity, minHeight: 48)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.startingLoading)
                    Text("Tap Start Loading to open the manifest for tap-check and per-order seal.")
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
                        onShowReDispatch: onShowReDispatch
                    )

                    if viewModel.allOrdersSealed && manifest.state != "SEALED" {
                        Button {
                            Task { await viewModel.sealManifest() }
                        } label: {
                            HStack {
                                Image(systemName: "lock.fill")
                                if viewModel.sealingManifest {
                                    ProgressView().controlSize(.regular)
                                } else {
                                    Text("Seal Manifest").font(.headline)
                                }
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

// MARK: - Order checklist

private struct OrderChecklistSection: View {
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            PayloadSectionHeader(
                title: "ORDER CHECKLIST",
                trailing: "\(viewModel.sealedOrderIds.count) / \(viewModel.orders.count) SEALED",
                trailingTint: viewModel.allOrdersSealed ? TermTheme.live : TermTheme.accent
            )
            .padding(.horizontal, 4)

            if viewModel.loadingOrders {
                PayloadStateView(
                    variant: .manifest,
                    title: "FETCHING_MANIFEST",
                    message: "Loading the checklist items assigned to this vehicle.",
                    compact: true
                )
                .frame(maxWidth: .infinity)
                .padding()
            } else if viewModel.orders.isEmpty {
                PayloadStateView(
                    variant: .manifest,
                    title: "NO_ORDERS_ASSIGNED",
                    message: "This truck has no checklist items waiting to load.",
                    compact: true
                )
                .frame(maxWidth: .infinity, alignment: .center)
                .padding(32)
                .tacticalCard()
            } else {
                VStack(spacing: 8) {
                    ForEach(viewModel.orders) { order in
                        OrderChip(
                            order: order,
                            selected: order.orderId == viewModel.selectedOrderId,
                            sealed: viewModel.sealedOrderIds.contains(order.orderId),
                            dispatchCode: viewModel.dispatchCodes[order.orderId],
                            onTap: { viewModel.selectOrder(order.orderId) }
                        )
                    }
                }

                if let selected = viewModel.orders.first(where: { $0.orderId == viewModel.selectedOrderId }) {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Text("LINE_ITEMS")
                                .font(.system(size: 12, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.secondary)
                            Spacer()
                            Text("ORD-\(selected.orderId.suffix(6).uppercased())")
                                .font(.system(size: 12, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.accent)
                        }
                        .padding(.top, 8)
                        
                        let items = selected.items ?? []
                        if items.isEmpty {
                            Text("NO_ITEMS_IN_ORDER")
                                .font(.system(size: 12, weight: .bold, design: .monospaced))
                                .foregroundStyle(TermTheme.tertiary)
                        } else {
                            VStack(spacing: 2) {
                                ForEach(items) { item in
                                    ItemRow(
                                        checked: viewModel.checkedItems.contains(item.lineItemId),
                                        enabled: !viewModel.sealedOrderIds.contains(selected.orderId),
                                        label: item.skuName.isEmpty ? item.skuId : item.skuName,
                                        quantity: item.quantity,
                                        onToggle: { viewModel.toggleItem(item.lineItemId) }
                                    )
                                }
                            }
                        }

                        if viewModel.sealedOrderIds.contains(selected.orderId) {
                            HStack {
                                Image(systemName: "lock.fill")
                                Text("ORDER_SEALED: \(viewModel.dispatchCodes[selected.orderId] ?? "")")
                            }
                            .font(.system(size: 14, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.live)
                            .padding()
                            .frame(maxWidth: .infinity)
                            .background(TermTheme.live.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                        } else {
                            Button {
                                Task { await viewModel.sealSelectedOrder() }
                            } label: {
                                HStack(spacing: 12) {
                                    if viewModel.sealingOrderId == selected.orderId {
                                        ProgressView().controlSize(.regular)
                                    } else {
                                        Image(systemName: "lock.shield.fill")
                                            .font(.system(size: 20))
                                        Text("SEAL_ORDER")
                                            .font(.system(size: 16, weight: .black, design: .monospaced))
                                    }
                                }
                                .padding()
                                .frame(maxWidth: .infinity)
                                .background(viewModel.canSealOrder(selected.orderId) ? TermTheme.accent : TermTheme.tertiary.opacity(0.3))
                                .foregroundStyle(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                            }
                            .disabled(viewModel.sealingOrderId != nil || !viewModel.canSealOrder(selected.orderId))
                            .buttonStyle(.tactical)

                            HStack(spacing: 12) {
                                Button("REPORT_ISSUE") { onShowException(selected.orderId) }
                                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                                    .padding(.vertical, 10)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.warn.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(TermTheme.warn)
                                    .buttonStyle(.tactical)

                                Button("RE_DISPATCH") { onShowReDispatch(selected.orderId) }
                                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                                    .padding(.vertical, 10)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.accent.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                                    .foregroundStyle(TermTheme.accent)
                                    .buttonStyle(.tactical)
                            }
                        }
                    }
                }
            }
        }
    }
}

private struct OrderChip: View {
    let order: LiveOrder
    let selected: Bool
    let sealed: Bool
    let dispatchCode: String?
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 12) {
                if sealed {
                    Image(systemName: "checkmark.seal.fill") // Tactical seal icon
                        .font(.system(size: 20))
                        .foregroundStyle(TermTheme.live)
                } else {
                    Image(systemName: "circle.dotted")
                        .font(.system(size: 20))
                        .foregroundStyle(selected ? TermTheme.accent : TermTheme.tertiary)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("ORD-\(order.orderId.suffix(6).uppercased())")
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                    
                    Text("\((order.items ?? []).count) UNITS")
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                Spacer()
                
                if sealed, let code = dispatchCode, !code.isEmpty {
                    Text(code)
                        .font(.system(size: 16, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.live)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(TermTheme.live.opacity(0.12), in: Capsule())
                }
            }
            .padding(14)
            .background {
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(background)
                    .overlay {
                        if selected {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.accent.opacity(0.2), lineWidth: 2)
                        } else {
                            RoundedRectangle(cornerRadius: 16, style: .continuous)
                                .stroke(TermTheme.separator.opacity(0.08), lineWidth: 1)
                        }
                    }
            }
        }
        .buttonStyle(.tactical)
    }

    private var background: Color {
        if sealed { return TermTheme.live.opacity(0.04) }
        if selected { return TermTheme.accent.opacity(0.08) }
        return TermTheme.card
    }
}

private struct ItemRow: View {
    let checked: Bool
    let enabled: Bool
    let label: String
    let quantity: Int
    let onToggle: () -> Void

    var body: some View {
        Button(action: { if enabled { onToggle() } }) {
            HStack(spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(checked ? TermTheme.accent : TermTheme.accent.opacity(0.06))
                        .frame(width: 32, height: 32)
                    
                    if checked {
                        Image(systemName: "checkmark")
                            .font(.system(size: 14, weight: .black))
                            .foregroundStyle(TermTheme.card)
                    }
                }
                
                Text(label.uppercased())
                    .font(.system(size: 13, weight: .bold, design: .monospaced))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .foregroundStyle(checked ? TermTheme.accent : TermTheme.secondary)
                
                Text("QTY: \(quantity)")
                    .font(.system(size: 13, weight: .black, design: .monospaced))
                    .foregroundStyle(TermTheme.accent)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 4)
                    .background(TermTheme.accent.opacity(0.06), in: RoundedRectangle(cornerRadius: 6))
            }
            .padding(.vertical, 8)
            .padding(.horizontal, 12)
            .background {
                if checked {
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .fill(TermTheme.accent.opacity(0.03))
                }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.tactical)
        .opacity(enabled ? 1 : 0.5)
        .disabled(!enabled)
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
                Text("ORD-\(orderId.suffix(6).uppercased())")
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
                Text("DOUBLE_CHECK_WINDOW: \(secondsLeft)s")
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
                                Text("ORD-\(id.suffix(6).uppercased())")
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

// MARK: - Phase 5 sheets

private struct InjectOrderSheet: View {
    let injecting: Bool
    let onCancel: () -> Void
    let onSubmit: (String) -> Void
    @State private var orderId: String = ""

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
                    
                    TextField("ORD-XXXXXX", text: $orderId)
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
                    
                    Text("Add an order mid-load. Dispatch will recompute the manifest.")
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                        .padding(.horizontal, 4)
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
                        Text("REMOVE_ORD-\(orderId.suffix(6).uppercased())")
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
                    
                    Text("3+ overflow attempts on this manifest will escalate to admin DLQ.")
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
    let onPick: (String) -> Void

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
                        Text("RE_DISPATCH_ORD-\(orderId.suffix(6).uppercased())")
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
                                                Button {
                                                    onPick(rec.driverId)
                                                } label: {
                                                    RecommendationRow(rec: rec)
                                                }
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
                    Text("REASSIGNING_ORDER...")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                }
            }
        }
    }
}

private struct RecommendationRow: View {
    let rec: TruckRecommendation

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(rec.driverName?.uppercased() ?? "ID-\(rec.driverId.prefix(8).uppercased())")
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                    
                    let meta = [rec.licensePlate, rec.vehicleClass, rec.truckStatus]
                        .compactMap { ($0?.isEmpty == false) ? $0 : nil }
                        .joined(separator: " • ")
                        .uppercased()
                    
                    if !meta.isEmpty {
                        Text(meta)
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                    }
                }
                
                Spacer()
                
                VStack(alignment: .trailing, spacing: 4) {
                    Text("OPTIMIZATION_SCORE")
                        .font(.system(size: 8, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.2f", rec.score ?? 0))
                        .font(.system(size: 16, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.live)
                }
            }
            
            HStack(spacing: 16) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("EST_TRAVEL")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f KM", rec.distanceKm ?? 0))
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                VStack(alignment: .leading, spacing: 2) {
                    Text("FREE_CAP")
                        .font(.system(size: 8, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.tertiary)
                    Text(String(format: "%.1f VU", rec.freeVolumeVu ?? 0))
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                
                Spacer()
                
                Image(systemName: "chevron.right.square.fill")
                    .font(.system(size: 24))
                    .foregroundStyle(TermTheme.accent)
            }
        }
        .padding(16)
        .background(TermTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .tacticalCard()
    }
}

// MARK: - Phase 6: notifications sheet / info banner

private struct NotificationsSheet: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.notifications.isEmpty {
                    PayloadStateView(
                        variant: .notifications,
                        title: "NO_NOTIFICATIONS",
                        message: "New events appear here in real time.",
                        compact: false
                    )
                } else {
                    List {
                        ForEach(viewModel.notifications) { n in
                            NotificationRow(item: n) {
                                if n.isUnread { viewModel.markNotificationRead(n.notificationId) }
                            }
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Notifications")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close") { viewModel.toggleNotificationsPanel() }
                }
                if viewModel.unreadCount > 0 {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("Mark all read") { viewModel.markAllNotificationsRead() }
                    }
                }
            }
        }
    }
}

private struct ManifestExceptionsSheet: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.loadingExceptions && viewModel.manifestExceptions.isEmpty {
                    PayloadLoadingView(
                        title: "LOADING EXCEPTIONS",
                        message: "Fetching overflow, damaged, and manual removals."
                    )
                } else if viewModel.manifestExceptions.isEmpty {
                    PayloadStateView(
                        variant: .warning,
                        title: "NO_EXCEPTIONS",
                        message: "Overflow, damaged, and manual removals appear here.",
                        compact: false
                    )
                } else {
                    List(viewModel.manifestExceptions) { row in
                        VStack(alignment: .leading, spacing: TermTheme.s8) {
                            HStack(spacing: TermTheme.s8) {
                                PayloadStatusBadge(text: row.reason)
                                if row.escalated {
                                    PayloadStatusBadge(text: "ESCALATED", tint: TermTheme.alert)
                                }
                            }
                            Text("Order \(row.orderId.prefix(8)) · Manifest \(row.manifestId.prefix(8))")
                                .font(.system(size: 13, weight: .medium, design: .monospaced))
                                .foregroundStyle(TermTheme.secondary)
                            Text("Attempts \(row.attemptCount)")
                                .font(.system(size: 11, weight: .bold, design: .monospaced))
                                .foregroundStyle(TermTheme.tertiary)
                        }
                        .padding(.vertical, TermTheme.s4)
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Manifest exceptions")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close") { viewModel.toggleExceptionsPanel() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Task { await viewModel.loadManifestExceptions() }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
        }
    }
}

private struct NotificationRow: View {
    let item: NotificationItem
    let onTap: () -> Void
    var body: some View {
        Button(action: onTap) {
            HStack(alignment: .top, spacing: 12) {
                Circle()
                    .fill(item.isUnread ? Color.accentColor : Color.clear)
                    .frame(width: 8, height: 8)
                    .padding(.top, 6)
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.title.isEmpty ? item.type : item.title)
                        .font(.headline)
                    if !item.body.isEmpty {
                        Text(item.body).font(.subheadline).foregroundStyle(.secondary)
                    }
                    if !item.createdAt.isEmpty {
                        Text(item.createdAt).font(.caption2).foregroundStyle(.tertiary)
                    }
                }
                Spacer()
            }
            .padding(.vertical, 4)
        }
        .buttonStyle(.tactical)
    }
}

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
