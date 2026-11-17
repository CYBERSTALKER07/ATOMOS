import SwiftUI

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
            VStack(alignment: .leading, spacing: 24) {
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
                        PegasusHaptics.medium()
                        Task { await viewModel.startLoading() }
                    } label: {
                        if viewModel.startingLoading {
                            ProgressView().controlSize(.regular)
                                .frame(maxWidth: .infinity, minHeight: 48)
                        } else {
                            Text("mobile_payload.ui.start_loading")
                                .font(.headline)
                                .frame(maxWidth: .infinity, minHeight: 48)
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.startingLoading)
                    .accessibilityLabel("Start loading manifest")

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
                            PegasusHaptics.heavy()
                            Task { await viewModel.sealManifest() }
                        } label: {
                            HStack {
                                Image(systemName: "lock.fill")
                                if viewModel.sealingManifest {
                                    ProgressView().controlSize(.regular)
                                } else {
                                    Text("mobile_payload.ui.seal_manifest").font(.headline)
                                }
                            }
                            .frame(maxWidth: .infinity, minHeight: 48)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.green)
                        .disabled(viewModel.sealingManifest)
                        .accessibilityLabel("Seal completed manifest")
                    }
                }
            }
            .padding()
        }
        .scrollDismissesKeyboard(.interactively)
    }
}
