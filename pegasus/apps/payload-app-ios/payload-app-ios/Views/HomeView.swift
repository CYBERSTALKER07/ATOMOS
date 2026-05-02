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
    @State private var viewModel = HomeViewModel()
    @State private var columnVisibility: NavigationSplitViewVisibility = .all
    @State private var showInjectSheet = false
    @State private var exceptionTargetOrderId: String?

    var body: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            TruckSidebar(viewModel: viewModel)
                .navigationTitle("Vehicles")
                .toolbar {
                    ToolbarItem(placement: .topBarLeading) {
                        OnlineDot(online: viewModel.online, queued: viewModel.queuedActions)
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
        .overlay(alignment: .bottom) {
            VStack(spacing: 8) {
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
        .onDisappear { viewModel.disconnectPhase6() }
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
                ProgressView().controlSize(.large)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if viewModel.trucks.isEmpty {
                ContentUnavailableView(
                    "No vehicles",
                    systemImage: "truck.box",
                    description: Text("Pull to refresh once dispatch assigns trucks.")
                )
            } else {
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
                ContentUnavailableView(
                    "Select a vehicle",
                    systemImage: "arrow.left",
                    description: Text("Pick a truck from the sidebar to load its manifest.")
                )
            } else if viewModel.manifestSealed {
                AllSealedSuccessView(
                    dispatchCodes: viewModel.dispatchCodes,
                    onStartNew: { Task { await viewModel.startNewManifest() } }
                )
            } else if viewModel.loadingManifest && viewModel.manifest == nil {
                ProgressView().controlSize(.large)
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
                ContentUnavailableView(
                    "No open manifest",
                    systemImage: "tray",
                    description: Text("This vehicle has no DRAFT or LOADING manifest. Wait for dispatch.")
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
                
                HStack(spacing: 20) {
                    StateBadge(state: manifest.state)
                    
                    VStack(alignment: .leading, spacing: 4) {
                        Text("PAYLOAD_VOLUME")
                            .font(.system(size: 10, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.tertiary)
                        Text(volumeLabel)
                            .font(.system(size: 16, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.accent)
                        
                        ProgressView(value: progress)
                            .tint(TermTheme.accent)
                            .scaleEffect(x: 1, y: 1.5, anchor: .center)
                    }
                    .padding(TermTheme.s20)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .tacticalCard()
                }

                HStack(spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("TARGET_STOPS")
                            .font(.system(size: 10, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.tertiary)
                        Text("\(manifest.stopCount ?? 0) UNITS")
                            .font(.system(size: 20, weight: .black, design: .monospaced))
                            .foregroundStyle(TermTheme.accent)
                    }
                    .padding(TermTheme.s20)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .tacticalCard()

                    if let region = manifest.regionCode, !region.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("DEPLOYMENT_ZONE")
                                .font(.system(size: 10, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.tertiary)
                            Text(region.uppercased())
                                .font(.system(size: 20, weight: .black, design: .monospaced))
                                .foregroundStyle(TermTheme.accent)
                        }
                        .padding(TermTheme.s20)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .tacticalCard()
                    }
                }

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
                            onDismiss: { viewModel.dismissCountdown() }
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

    private var progress: Double {
        let cap = max(manifest.maxVolumeVu ?? 0, 0.001)
        let total = manifest.totalVolumeVu ?? 0
        return min(max(total / cap, 0), 1)
    }

    private var volumeLabel: String {
        String(format: "%.1f / %.1f VU", manifest.totalVolumeVu ?? 0, manifest.maxVolumeVu ?? 0)
    }
}

// MARK: - Order checklist

private struct OrderChecklistSection: View {
    @Bindable var viewModel: HomeViewModel
    let onShowException: (String) -> Void
    let onShowReDispatch: (String) -> Void

    var body: some View {
        GroupBox("Orders (\(viewModel.sealedOrderIds.count)/\(viewModel.orders.count) sealed)") {
            if viewModel.loadingOrders {
                ProgressView().padding()
            } else if viewModel.orders.isEmpty {
                Text("No LOADED orders for this vehicle yet. They appear once dispatch assigns them.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .padding()
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
                    Divider().padding(.vertical, 8)
                    Text("Items — \(String(selected.orderId.prefix(8)))")
                        .font(.headline)
                    let items = selected.items ?? []
                    if items.isEmpty {
                        Text("No line items on this order.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    } else {
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
                    if viewModel.sealedOrderIds.contains(selected.orderId) {
                        Text("Order sealed. Dispatch code \(viewModel.dispatchCodes[selected.orderId] ?? "").")
                            .font(.footnote)
                            .foregroundStyle(.green)
                    } else {
                        Button {
                            Task { await viewModel.sealSelectedOrder() }
                        } label: {
                            HStack {
                                Image(systemName: "lock.fill")
                                if viewModel.sealingOrderId == selected.orderId {
                                    ProgressView().controlSize(.regular)
                                } else {
                                    Text("Seal Order").font(.headline)
                                }
                            }
                            .frame(maxWidth: .infinity, minHeight: 44)
                        }
                        .buttonStyle(.bordered)
                        .tint(.blue)
                        .disabled(!viewModel.canSealOrder(selected.orderId) || viewModel.sealingOrderId == selected.orderId)

                        HStack(spacing: 8) {
                            Button(role: .destructive) {
                                onShowException(selected.orderId)
                            } label: {
                                HStack {
                                    Image(systemName: "exclamationmark.triangle.fill")
                                    if viewModel.exceptionLoadingOrderId == selected.orderId {
                                        ProgressView().controlSize(.small)
                                    } else {
                                        Text("Remove").font(.subheadline.weight(.medium))
                                    }
                                }
                                .frame(maxWidth: .infinity, minHeight: 40)
                            }
                            .buttonStyle(.bordered)
                            .disabled(viewModel.exceptionLoadingOrderId == selected.orderId)

                            Button {
                                onShowReDispatch(selected.orderId)
                            } label: {
                                HStack {
                                    Image(systemName: "arrow.left.arrow.right")
                                    Text("Re-Dispatch").font(.subheadline.weight(.medium))
                                }
                                .frame(maxWidth: .infinity, minHeight: 40)
                            }
                            .buttonStyle(.bordered)
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
        .buttonStyle(.plain)
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
            HStack {
                Image(systemName: checked ? "checkmark.square.fill" : "square")
                    .foregroundStyle(checked ? .blue : .secondary)
                Text(label)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Text("x\(quantity)").font(.body.weight(.medium))
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .opacity(enabled ? 1 : 0.5)
        .disabled(!enabled)
    }
}

// MARK: - Post-seal 60s countdown

private struct PostSealCountdownView: View {
    let orderId: String
    let dispatchCode: String
    let secondsLeft: Int
    let onDismiss: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Order \(String(orderId.prefix(8))) sealed")
                .font(.headline)
            Text("Dispatch code").font(.caption).foregroundStyle(.secondary)
            Text(dispatchCode)
                .font(.largeTitle.monospaced().weight(.bold))
                .foregroundStyle(.green)
            Text("Double-check window: \(secondsLeft)s").font(.subheadline)
            ProgressView(value: Double(secondsLeft) / 60.0)
            Button("Continue", action: onDismiss)
                .buttonStyle(.bordered)
        }
        .padding()
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.green.opacity(0.10), in: RoundedRectangle(cornerRadius: 16))
    }
}

// MARK: - All Sealed success terminal screen

private struct AllSealedSuccessView: View {
    let dispatchCodes: [String: String]
    let onStartNew: () -> Void

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                Image(systemName: "checkmark.seal.fill")
                    .resizable()
                    .scaledToFit()
                    .frame(width: 72, height: 72)
                    .foregroundStyle(.green)
                Text("Manifest Sealed").font(.title.weight(.bold))
                Text("\(dispatchCodes.count) order\(dispatchCodes.count == 1 ? "" : "s") dispatched")
                    .foregroundStyle(.secondary)

                if !dispatchCodes.isEmpty {
                    GroupBox("Dispatch Codes") {
                        VStack(spacing: 6) {
                            ForEach(dispatchCodes.sorted(by: { $0.key < $1.key }), id: \.key) { (orderId, code) in
                                HStack {
                                    Text(String(orderId.prefix(8))).font(.footnote)
                                    Spacer()
                                    Text(code).font(.callout.monospaced().weight(.bold))
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }

                Button(action: onStartNew) {
                    Text("Start New Manifest")
                        .font(.headline)
                        .frame(maxWidth: .infinity, minHeight: 48)
                }
                .buttonStyle(.borderedProminent)
            }
            .padding()
        }
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

private struct StateBadge: View {
    let state: String
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("PROTOCOL_STATE")
                .font(.system(size: 10, weight: .black, design: .monospaced))
                .foregroundStyle(TermTheme.tertiary)
                .tracking(1.4)
            
            Text(state.uppercased())
                .font(.system(size: 24, weight: .black, design: .monospaced))
                .foregroundStyle(statusColor)
                .tracking(2.0)
        }
        .padding(TermTheme.s24)
        .frame(minWidth: 200, alignment: .leading)
        .tacticalCard()
    }

    private var statusColor: Color {
        switch state.uppercased() {
        case "LOADING": return TermTheme.progress
        case "SEALED", "DISPATCHED": return TermTheme.live
        case "DRAFT": return TermTheme.warn
        default: return TermTheme.accent
        }
    }
}
}

// MARK: - Phase 5 sheets

private struct InjectOrderSheet: View {
    let injecting: Bool
    let onCancel: () -> Void
    let onSubmit: (String) -> Void
    @State private var orderId: String = ""

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Order ID", text: $orderId)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .disabled(injecting)
                } footer: {
                    Text("Add an order mid-load. Dispatch will recompute the manifest.")
                }
            }
            .navigationTitle("Inject Order")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel", action: onCancel).disabled(injecting)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        onSubmit(orderId)
                    } label: {
                        if injecting { ProgressView() } else { Text("Inject") }
                    }
                    .disabled(injecting || orderId.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
    }
}

private struct ExceptionReasonSheet: View {
    let orderId: String
    let inFlight: Bool
    let onCancel: () -> Void
    let onSelect: (String) -> Void

    private let reasons: [(code: String, label: String)] = [
        ("OVERFLOW", "Overflow — no space"),
        ("DAMAGED", "Damaged goods"),
        ("MANUAL", "Manual exception"),
    ]

    var body: some View {
        NavigationStack {
            List {
                Section {
                    ForEach(reasons, id: \.code) { reason in
                        Button {
                            onSelect(reason.code)
                        } label: {
                            HStack {
                                Text(reason.label).foregroundStyle(.primary)
                                Spacer()
                                if inFlight {
                                    ProgressView().controlSize(.small)
                                }
                            }
                        }
                        .disabled(inFlight)
                    }
                } header: {
                    Text("Reason")
                } footer: {
                    Text("3+ overflow attempts on this manifest will escalate to admin DLQ.")
                }
            }
            .navigationTitle("Remove \(String(orderId.prefix(8)))")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel", action: onCancel).disabled(inFlight)
                }
            }
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
        NavigationStack {
            Group {
                if loading {
                    ProgressView().controlSize(.large)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let resp = response {
                    List {
                        Section {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(resp.retailerName?.isEmpty == false ? resp.retailerName! : "Order")
                                    .font(.headline)
                                Text(String(format: "%.1f VU", resp.orderVolumeVu ?? 0))
                                    .font(.subheadline.monospacedDigit())
                                    .foregroundStyle(.secondary)
                            }
                        }
                        if (resp.recommendations).isEmpty {
                            Section {
                                Text("No suitable trucks. Try again later or remove the order.")
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                            }
                        } else {
                            Section("Recommendations") {
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
                } else {
                    ContentUnavailableView(
                        "No recommendations",
                        systemImage: "tray",
                        description: Text("No recommendations available.")
                    )
                }
            }
            .navigationTitle("Re-Dispatch \(String(orderId.prefix(8)))")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    if reassigning {
                        ProgressView()
                    } else {
                        Button("Close", action: onClose)
                    }
                }
            }
        }
    }
}

private struct RecommendationRow: View {
    let rec: TruckRecommendation

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(rec.driverName?.isEmpty == false ? rec.driverName! : String(rec.driverId.prefix(8)))
                    .font(.subheadline.weight(.semibold))
                Spacer()
                Text(String(format: "score %.2f", rec.score ?? 0))
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
            }
            Text(meta).font(.caption).foregroundStyle(.secondary)
            Text(stats).font(.caption.monospacedDigit())
            if let r = rec.recommendation, !r.isEmpty {
                Text(r).font(.caption2).foregroundStyle(.tint)
            }
        }
        .padding(.vertical, 2)
    }

    private var meta: String {
        [rec.licensePlate, rec.vehicleClass, rec.truckStatus]
            .compactMap { ($0?.isEmpty == false) ? $0 : nil }
            .joined(separator: " • ")
    }

    private var stats: String {
        String(
            format: "%.1f km • free %.1f VU • %d orders",
            rec.distanceKm ?? 0,
            rec.freeVolumeVu ?? 0,
            rec.orderCount ?? 0
        )
    }
}

// MARK: - Phase 6: Online chip / notifications sheet / info banner

private struct OnlineDot: View {
    let online: Bool
    let queued: Int
    var body: some View {
        HStack(spacing: 6) {
            Circle()
                .fill(online ? Color.green : Color.red)
                .frame(width: 8, height: 8)
            Text(online ? "Live" : queued > 0 ? "Offline · \(queued) queued" : "Offline")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}

private struct NotificationsSheet: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.notifications.isEmpty {
                    ContentUnavailableView(
                        "No notifications",
                        systemImage: "bell.slash",
                        description: Text("New events appear here in real time.")
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
        .buttonStyle(.plain)
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
