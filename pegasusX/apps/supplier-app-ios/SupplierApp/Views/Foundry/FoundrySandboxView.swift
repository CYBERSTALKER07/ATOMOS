//
//  FoundrySandboxView.swift
//  SupplierAppIOS
//

import SwiftUI

struct FoundrySandboxView: View {

    @State private var vm = FoundrySandboxViewModel()
    @State private var selectedTab: Int = 0
    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Section", selection: $selectedTab) {
                    Text("Scenario Sandbox (CoW)").tag(0)
                    Text("Lot Recall Blast-Radius").tag(1)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 16)
                .padding(.vertical, 8)

                if let err = vm.errorMessage {
                    HStack {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundStyle(.red)
                        Text(err)
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(.red)
                    }
                    .padding(12)
                    .background(Color.red.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                    .padding(.horizontal, 16)
                }

                if let msg = vm.statusMessage {
                    HStack {
                        Image(systemName: vm.hasConflict ? "exclamationmark.octagon.fill" : "checkmark.circle.fill")
                            .foregroundStyle(vm.hasConflict ? .red : .green)
                        Text(msg)
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(vm.hasConflict ? .red : .green)
                    }
                    .padding(12)
                    .background((vm.hasConflict ? Color.red : Color.green).opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 10))
                    .padding(.horizontal, 16)
                }

                if selectedTab == 0 {
                    scenarioSandboxTab
                } else {
                    lotRecallTab
                }
            }
            .navigationTitle("Foundry Operations")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(action: onBack) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 16, weight: .semibold))
                    }
                }
            }
        }
    }

    private var scenarioSandboxTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Isolation Card
                VStack(alignment: .leading, spacing: 8) {
                    Text("Scenario Isolation Layer")
                        .font(.system(size: 15, weight: .bold))
                    Text("Copy-on-Write (CoW) creates an isolated branch to simulate price, lot, and policy changes without mutating production data.")
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                    TextField("Scenario ID", text: $vm.scenarioId)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(size: 13, design: .monospaced))
                }
                .padding(14)
                .background(Color(.secondarySystemGroupedBackground))
                .clipShape(RoundedRectangle(cornerRadius: 14))

                // Patch Editor
                VStack(alignment: .leading, spacing: 10) {
                    Text("Add CoW Patch")
                        .font(.system(size: 15, weight: .bold))

                    HStack(spacing: 8) {
                        TextField("Object Type", text: $vm.objectType)
                            .textFieldStyle(.roundedBorder)
                            .font(.system(size: 13))
                        TextField("Object ID", text: $vm.objectId)
                            .textFieldStyle(.roundedBorder)
                            .font(.system(size: 13, design: .monospaced))
                    }

                    HStack(spacing: 8) {
                        TextField("Property Key", text: $vm.patchKey)
                            .textFieldStyle(.roundedBorder)
                            .font(.system(size: 13))
                        TextField("New Value", text: $vm.patchValue)
                            .textFieldStyle(.roundedBorder)
                            .font(.system(size: 13, design: .monospaced))
                    }

                    Button {
                        Task { await vm.applyCowPatch() }
                    } label: {
                        HStack {
                            Image(systemName: "plus.circle.fill")
                            Text("Apply Patch to Branch")
                        }
                        .font(.system(size: 14, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .frame(height: 40)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                }
                .padding(14)
                .background(Color(.secondarySystemGroupedBackground))
                .clipShape(RoundedRectangle(cornerRadius: 14))

                // Staged Patches
                if !vm.appliedPatches.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Staged CoW Patches (\(vm.appliedPatches.count))")
                            .font(.system(size: 14, weight: .semibold))

                        ForEach(vm.appliedPatches) { patch in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(patch.objectType):\(patch.objectId)")
                                        .font(.system(size: 13, weight: .bold))
                                    Text("\(patch.key) → \(patch.value)")
                                        .font(.system(size: 12, design: .monospaced))
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text("STAGED")
                                    .font(.system(size: 10, weight: .heavy))
                                    .foregroundStyle(Color.blue)
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 3)
                                    .background(Color.blue.opacity(0.12))
                                    .clipShape(Capsule())
                            }
                            .padding(10)
                            .background(Color(.tertiarySystemGroupedBackground))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                }

                // Conflicts
                if !vm.conflicts.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Concurrency Conflicts (409)")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(.red)

                        ForEach(vm.conflicts) { conf in
                            VStack(alignment: .leading, spacing: 4) {
                                Text("\(conf.objectType):\(conf.objectId)")
                                    .font(.system(size: 13, weight: .bold))
                                Text("Base Version: \(conf.baseVersion) | Live Version: \(conf.liveVersion)")
                                    .font(.system(size: 11, design: .monospaced))
                                Text("Baseline changed concurrently. Rebase required before commit.")
                                    .font(.system(size: 10, weight: .semibold))
                                    .foregroundStyle(.red)
                            }
                            .padding(10)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.red.opacity(0.1))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                }

                // Commit Button
                Button {
                    Task { await vm.commitScenario() }
                } label: {
                    HStack {
                        if vm.isCommitting {
                            ProgressView().tint(.white)
                            Spacer().frame(width: 8)
                        }
                        Text("Optimistic Commit to Live Production Layer")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .frame(maxWidth: .infinity)
                    .frame(height: 48)
                    .background(Color.primary)
                    .foregroundStyle(Color(.systemBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                }
                .disabled(vm.isCommitting)

                Spacer(minLength: 40)
            }
            .padding(16)
        }
    }

    private var lotRecallTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Contaminated Lot Blast-Radius")
                        .font(.system(size: 15, weight: .bold))
                    Text("Trace forward lineage of defective raw material lots across all downstream work orders, pallets, and in-transit deliveries.")
                        .font(.system(size: 12))
                        .foregroundStyle(.secondary)
                    TextField("Raw Material Lot Barcode", text: $vm.lotBarcode)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(size: 13, design: .monospaced))

                    Button {
                        Task { await vm.analyzeLotRecall() }
                    } label: {
                        HStack {
                            if vm.isAnalyzingRecall {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text("Trace Blast Radius Lineage")
                                .font(.system(size: 14, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 42)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                    .disabled(vm.isAnalyzingRecall)
                }
                .padding(14)
                .background(Color(.secondarySystemGroupedBackground))
                .clipShape(RoundedRectangle(cornerRadius: 14))

                if let report = vm.recallReport {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Forward Blast Radius Results")
                            .font(.system(size: 15, weight: .bold))

                        MetricCard(title: "Affected Work Orders", count: report.affectedWorkOrders.count, items: report.affectedWorkOrders)
                        MetricCard(title: "Affected Finished Lots", count: report.affectedFinishedLots.count, items: report.affectedFinishedLots)
                        MetricCard(title: "Quarantined SSCC Pallets", count: report.quarantinedSsccPallets.count, items: report.quarantinedSsccPallets, isWarning: true)
                        MetricCard(title: "Quarantined Warehouse Totes", count: report.quarantinedWarehouseTotes.count, items: report.quarantinedWarehouseTotes)
                        MetricCard(title: "In-Transit Driver Drop Blocks", count: report.inTransitDriverDropBlocks.count, items: report.inTransitDriverDropBlocks, isWarning: true)
                        MetricCard(title: "Impacted Soliq E-Facturas", count: report.impactedSoliqFacturas.count, items: report.impactedSoliqFacturas)

                        Button {
                            Task { await vm.executeQuarantineBlastRadius() }
                        } label: {
                            HStack {
                                if vm.isQuarantining {
                                    ProgressView().tint(.white)
                                    Spacer().frame(width: 8)
                                }
                                Text(vm.quarantineSuccess ? "QUARANTINE ENFORCED" : "LOCKDOWN & QUARANTINE BLAST RADIUS")
                                    .font(.system(size: 15, weight: .bold))
                            }
                            .frame(maxWidth: .infinity)
                            .frame(height: 50)
                            .background(Color.red)
                            .foregroundStyle(.white)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .disabled(vm.isQuarantining || vm.quarantineSuccess)
                    }
                }

                Spacer(minLength: 40)
            }
            .padding(16)
        }
    }
}

private struct MetricCard: View {
    let title: String
    let count: Int
    let items: [String]
    var isWarning: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(title)
                    .font(.system(size: 13, weight: .semibold))
                Spacer()
                Text("\(count)")
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(isWarning ? Color.red : Color.primary)
            }
            if !items.isEmpty {
                Text(items.joined(separator: ", "))
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
        }
        .padding(10)
        .background(isWarning ? Color.red.opacity(0.08) : Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}
