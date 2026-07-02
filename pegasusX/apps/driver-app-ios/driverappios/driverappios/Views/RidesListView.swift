//
//  RidesListView.swift
//  driverappios
//

import CoreLocation
import SwiftUI

/// Tab 2: "Rides" — upcoming routes with full order details, premium card UI
struct RidesListView: View {
    @Bindable var vm: FleetViewModel
    @State private var driverSocketState = DriverSocketState.shared
    @State private var loadingMode = false
    @State private var showEarlyComplete = false
    @State private var isRequestingEarlyComplete = false
    @State private var earlyCompleteMessage: String?
    var onRequestEarlyComplete: ((String, String) -> Void)?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // MARK: - Header
                VStack(alignment: .leading, spacing: 6) {
                    Text(loadingMode ? "LOADING SEQUENCE" : "UPCOMING")
                        .font(.system(size: 10, weight: .heavy, design: .monospaced))
                        .foregroundStyle(loadingMode ? LabTheme.fg : LabTheme.fgTertiary)
                        .tracking(1.2)

                    HStack(alignment: .top) {
                        HStack(alignment: .firstTextBaseline, spacing: 10) {
                            Text(loadingMode ? "Loading Manifest" : "Route Manifest")
                                .font(.system(size: 28, weight: .bold))
                                .foregroundStyle(LabTheme.fg)

                            if !vm.pendingMissions.isEmpty {
                                Text("\(vm.pendingMissions.count)")
                                    .font(.system(size: 12, weight: .bold))
                                    .foregroundStyle(LabTheme.buttonFg)
                                    .frame(width: 24, height: 24)
                                    .background(LabTheme.fg, in: Circle())
                            }
                        }
                        Spacer()
                        VStack(alignment: .trailing, spacing: 4) {
                            Text("Loading Mode")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(loadingMode ? LabTheme.fg : LabTheme.fgTertiary)
                                .tracking(0.8)
                            Toggle("", isOn: $loadingMode)
                                .labelsHidden()
                                .tint(LabTheme.fg)
                        }
                    }
                }
                .padding(.horizontal, LabTheme.s20)
                .padding(.top, 60)
                .padding(.bottom, LabTheme.s20)

                // MARK: - Mission Cards

                // LEO: Ghost Stop Prevention banner
                if vm.awaitingSeal {
                    ExplainStatusBanner(
                        explain: vm.gateExplain,
                        fallbackTitle: "AWAITING SEAL",
                        fallbackDetail: "Manifest is \(vm.manifestState ?? "not sealed"). Payloader must complete loading and seal before you can depart."
                    )
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.bottom, 8)
                }

                if vm.isLoadingMissions {
                    loadingView
                } else if vm.pendingMissions.isEmpty {
                    emptyView
                } else {
                    let displayMissions = loadingMode ? Array(vm.pendingMissions.reversed()) : vm.pendingMissions
                    LazyVStack(spacing: 14) {
                        ForEach(Array(displayMissions.enumerated()), id: \.element.id) { index, mission in
                            HStack(spacing: 8) {
                                rideCard(mission, index: index, loadSeqLabel: loadSeqLabel(for: index, total: displayMissions.count))

                                if !loadingMode && displayMissions.count > 1 {
                                    VStack(spacing: 4) {
                                        Button {
                                            vm.moveOrder(from: index, to: index - 1)
                                        } label: {
                                            Image(systemName: "chevron.up")
                                                .font(.system(size: 14, weight: .bold))
                                                .foregroundStyle(index == 0 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 32, height: 32)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .disabled(index == 0)

                                        Button {
                                            vm.moveOrder(from: index, to: index + 1)
                                        } label: {
                                            Image(systemName: "chevron.down")
                                                .font(.system(size: 14, weight: .bold))
                                                .foregroundStyle(index == displayMissions.count - 1 ? LabTheme.fgTertiary : LabTheme.fg)
                                                .frame(width: 32, height: 32)
                                                .background(LabTheme.fg.opacity(0.06), in: Circle())
                                        }
                                        .disabled(index == displayMissions.count - 1)
                                    }
                                }
                            }
                        }
                    }
                    .padding(.horizontal, LabTheme.s16)
                }
            }
            .padding(.bottom, 20)
        }
        .scrollIndicators(.hidden)
        .background(LabTheme.bg)
        .refreshable {
            await vm.loadMissions()
        }
        .overlay(alignment: .bottomTrailing) {
            // Edge 27: Early Complete FAB
            if !vm.pendingMissions.isEmpty {
                Button {
                    showEarlyComplete = true
                } label: {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.system(size: 20, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 56, height: 56)
                        .background(LabTheme.destructive, in: Circle())
                        .shadow(color: LabTheme.destructive.opacity(0.3), radius: 8, y: 4)
                }
                .padding(24)
            }
        }
        .sheet(isPresented: $showEarlyComplete) {
            EarlyCompleteSheet(onConfirm: { reason, note in
                showEarlyComplete = false
                if let onRequestEarlyComplete {
                    onRequestEarlyComplete(reason, note)
                } else {
                    Task { await submitEarlyComplete(reason: reason, note: note) }
                }
            })
            .presentationDetents([.medium])
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            let hadInFlight = isRequestingEarlyComplete
            if isRequestingEarlyComplete {
                isRequestingEarlyComplete = false
                earlyCompleteMessage = DriverReconnectRecovery.hint
            }
            Task {
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                await vm.loadMissions()
            }
        }
        .safeAreaInset(edge: .bottom) {
            if let earlyCompleteMessage {
                Text(earlyCompleteMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }

    private func submitEarlyComplete(reason: String, note: String) async {
        isRequestingEarlyComplete = true
        earlyCompleteMessage = nil
        defer { isRequestingEarlyComplete = false }
        do {
            _ = try await APIClient.shared.requestEarlyComplete(reason: reason, note: note)
            earlyCompleteMessage = "Early complete request submitted."
            await vm.loadMissions()
        } catch {
            earlyCompleteMessage = error.localizedDescription
        }
    }

    // MARK: - Ride Card

    private func rideCard(_ mission: Mission, index: Int, loadSeqLabel: String? = nil) -> some View {
        Button {
            vm.selectMission(mission)
        } label: {
            VStack(alignment: .leading, spacing: 16) {
                // Loading sequence badge
                if let label = loadSeqLabel {
                    Text(label)
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.buttonFg)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(LabTheme.fg, in: RoundedRectangle(cornerRadius: 4))
                }

                // Top row: order id + status
                HStack {
                    Text(mission.order_id)
                        .font(.system(size: 15, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)

                    Spacer()

                    StatusPill(label: mission.state, color: LabTheme.fg)
                }

                // Amount row
                HStack(spacing: 12) {
                    infoChip(icon: "creditcard", text: mission.gateway)
                    infoChip(icon: "banknote", text: mission.amount.formattedAmount)
                }

                // Coordinates
                VStack(alignment: .leading, spacing: 6) {
                    Text("DELIVERY TARGET")
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)

                    Text(String(format: "%.4f, %.4f", mission.target_lat, mission.target_lng))
                        .font(.system(size: 14, weight: .semibold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                }

                // Distance + action
                HStack {
                    distanceInfo(mission)
                    Spacer()
                    Image(systemName: "arrow.right")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                        .padding(8)
                        .background(LabTheme.fg.opacity(0.08), in: Circle())
                }

                // Bottom accent bar
                RoundedRectangle(cornerRadius: 2)
                    .fill(LabTheme.fg.opacity(0.12))
                    .frame(height: 2)
            }
            .padding(LabTheme.s20)
            .labCard()
        }
        .buttonStyle(.pressable)
        .staggeredAppear(index: index)
    }

    // MARK: - Load Sequence Label

    private func loadSeqLabel(for index: Int, total: Int) -> String? {
        guard loadingMode else { return nil }
        if index == 0 { return "Load #\(index + 1) · Back of Truck" }
        if index == total - 1 { return "Load #\(index + 1) · By the Doors" }
        return "Load #\(index + 1)"
    }

    // MARK: - Info Chip

    private func infoChip(icon: String, text: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 10, weight: .semibold))
            Text(text)
                .font(.system(size: 12, weight: .medium))
        }
        .foregroundStyle(LabTheme.fgSecondary)
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(LabTheme.fg.opacity(0.04), in: Capsule())
    }

    // MARK: - Distance Info

    @ViewBuilder
    private func distanceInfo(_ mission: Mission) -> some View {
        if let loc = vm.location {
            let target = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
            let dist = haversineDistance(from: loc, to: target)
            let inRange = dist <= 500_000

            HStack(spacing: 6) {
                Circle()
                    .fill(inRange ? LabTheme.success : LabTheme.fgTertiary)
                    .frame(width: 6, height: 6)

                Text(formattedDistance(dist))
                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                    .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
            }
        }
    }

    // MARK: - Loading

    private var loadingView: some View {
        DriverLoadingView(
            title: "Loading routes",
            message: "Checking manifest state, sequence, and delivery assignments."
        )
        .frame(maxWidth: .infinity)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, 60)
    }

    // MARK: - Empty

    private var emptyView: some View {
        DriverEmptyView(
            icon: "road.lanes",
            title: "No upcoming rides",
            message: "Pull to refresh or check back later."
        )
        .frame(maxWidth: .infinity)
        .padding(.horizontal, LabTheme.s16)
        .padding(.vertical, 60)
    }
}

// MARK: - Edge 27: Early Complete Sheet

private struct EarlyCompleteSheet: View {
    let onConfirm: (String, String) -> Void
    @Environment(\.dismiss) private var dismiss

    @State private var selectedReason = "FATIGUE"
    @State private var note = ""

    private let reasons: [(id: String, label: String)] = [
        ("FATIGUE", "Fatigue / Feeling Unwell"),
        ("TRAFFIC", "Heavy Traffic / Road Block"),
        ("VEHICLE_ISSUE", "Vehicle Issue"),
        ("OTHER", "Other")
    ]

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Text("Remaining orders will be returned to the supplier for next-day re-dispatch.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Section("Reason") {
                    ForEach(reasons, id: \.id) { reason in
                        Button {
                            selectedReason = reason.id
                        } label: {
                            HStack {
                                Text(reason.label)
                                    .foregroundStyle(LabTheme.fg)
                                Spacer()
                                if selectedReason == reason.id {
                                    Image(systemName: "checkmark")
                                        .foregroundStyle(.blue)
                                        .fontWeight(.semibold)
                                }
                            }
                        }
                    }
                }

                Section("Note (optional)") {
                    TextField("Add a note", text: $note, axis: .vertical)
                        .lineLimit(2...4)
                }
            }
            .navigationTitle("Request Early Complete")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") {
                        onConfirm(selectedReason, note)
                    }
                    .foregroundStyle(LabTheme.destructive)
                    .fontWeight(.semibold)
                }
            }
        }
    }
}

#Preview {
    RidesListView(vm: FleetViewModel())
}
