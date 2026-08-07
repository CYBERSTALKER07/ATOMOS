import SwiftUI

// MARK: - Detail

struct ManifestDetailPane: View {
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

struct ManifestWorkflow: View {
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

struct PostSealCountdownView: View {
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

struct AllSealedSuccessView: View {
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

struct TruckHeader: View {
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
