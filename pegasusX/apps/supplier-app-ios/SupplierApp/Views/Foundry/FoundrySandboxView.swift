//
//  FoundrySandboxView.swift
//  SupplierAppIOS
//

import SwiftUI

struct FoundrySandboxView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = FoundrySandboxViewModel()
    @State private var selectedTab: Int = 0
    var onBack: (() -> Void)? = nil

    init(onBack: (() -> Void)? = nil) {
        self.onBack = onBack
    }

    var body: some View {
        VStack(spacing: 0) {
            Picker("mobile_supplier.chrome.section", selection: $selectedTab) {
                Text("mobile_supplier.chrome.scenario_sandbox_cow").tag(0)
                Text("mobile_supplier.chrome.lot_recall_blast").tag(1)
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .onChange(of: selectedTab) { _, _ in
                PegasusHaptics.selection()
            }

            if let err = vm.errorMessage {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundStyle(SupplierTheme.destructive)
                    Text(err)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(SupplierTheme.destructive)
                }
                .padding(12)
                .background(SupplierTheme.destructive.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .padding(.horizontal, 16)
            }

            if let msg = vm.statusMessage {
                HStack {
                    Image(systemName: vm.hasConflict ? "exclamationmark.octagon.fill" : "checkmark.circle.fill")
                        .foregroundStyle(vm.hasConflict ? SupplierTheme.destructive : SupplierTheme.success)
                    Text(msg)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(vm.hasConflict ? SupplierTheme.destructive : SupplierTheme.success)
                }
                .padding(12)
                .background((vm.hasConflict ? SupplierTheme.destructive : SupplierTheme.success).opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10))
                .padding(.horizontal, 16)
            }

            if selectedTab == 0 {
                scenarioSandboxTab
            } else {
                lotRecallTab
            }
        }
        .navigationTitle("mobile_supplier.chrome.foundry_operations")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    PegasusHaptics.light()
                    if let onBack {
                        onBack()
                    } else {
                        dismiss()
                    }
                } label: {
                    Image(systemName: "chevron.left")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("common.action.back"))
            }
        }
    }

    private var scenarioSandboxTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                // Isolation Card
                VStack(alignment: .leading, spacing: 8) {
                    Text("mobile_supplier.chrome.scenario_isolation")
                        .font(.headline)
                    Text("mobile_supplier.chrome.cow_explainer")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("mobile_supplier.chrome.scenario_id", text: $vm.scenarioId)
                        .textFieldStyle(.roundedBorder)
                        .font(.subheadline.monospaced())
                }
                .padding(14)
                .background(SupplierTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 14))

                // Patch Editor
                VStack(alignment: .leading, spacing: 10) {
                    Text("mobile_supplier.chrome.add_cow_patch")
                        .font(.headline)

                    HStack(spacing: 8) {
                        TextField("mobile_supplier.chrome.object_type", text: $vm.objectType)
                            .textFieldStyle(.roundedBorder)
                            .font(.subheadline)
                        TextField("mobile_supplier.chrome.object_id", text: $vm.objectId)
                            .textFieldStyle(.roundedBorder)
                            .font(.subheadline.monospaced())
                    }

                    HStack(spacing: 8) {
                        TextField("mobile_supplier.chrome.property_key", text: $vm.patchKey)
                            .textFieldStyle(.roundedBorder)
                            .font(.subheadline)
                        TextField("mobile_supplier.chrome.new_value", text: $vm.patchValue)
                            .textFieldStyle(.roundedBorder)
                            .font(.subheadline.monospaced())
                    }

                    Button {
                        PegasusHaptics.light()
                        Task { await vm.applyCowPatch() }
                    } label: {
                        HStack {
                            Image(systemName: "plus.circle.fill")
                            Text("mobile_supplier.chrome.apply_patch_to_branch")
                        }
                        .font(.subheadline.bold())
                        .frame(maxWidth: .infinity)
                        .frame(minHeight: 44)
                        .background(SupplierTheme.primaryAction)
                        .foregroundStyle(SupplierTheme.onPrimaryAction)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                }
                .padding(14)
                .background(SupplierTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 14))

                // Staged Patches
                if !vm.appliedPatches.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(L10n.format("mobile_supplier.chrome.staged_cow_patches", "\(vm.appliedPatches.count)"))
                            .font(.subheadline.weight(.semibold))

                        ForEach(vm.appliedPatches) { patch in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(patch.objectType):\(patch.objectId)")
                                        .font(.subheadline.bold())
                                    Text("\(patch.key) → \(patch.value)")
                                        .font(.caption.monospaced())
                                        .foregroundStyle(SupplierTheme.secondaryLabel)
                                }
                                Spacer()
                                Text("STAGED")
                                    .font(.caption2.weight(.heavy))
                                    .foregroundStyle(SupplierTheme.primaryAction)
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 3)
                                    .background(SupplierTheme.fill)
                                    .clipShape(Capsule())
                            }
                            .padding(10)
                            .background(SupplierTheme.secondaryBackground)
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                }

                // Conflicts
                if !vm.conflicts.isEmpty {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("mobile_supplier.chrome.concurrency_conflicts")
                            .font(.subheadline.bold())
                            .foregroundStyle(SupplierTheme.destructive)

                        ForEach(vm.conflicts) { conf in
                            VStack(alignment: .leading, spacing: 4) {
                                Text("\(conf.objectType):\(conf.objectId)")
                                        .font(.subheadline.bold())
                                Text(L10n.format("mobile_supplier.chrome.base_vs_live_version", "\(conf.baseVersion)", "\(conf.liveVersion)"))
                                    .font(.caption.monospaced())
                                Text("mobile_supplier.chrome.baseline_changed")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(SupplierTheme.destructive)
                            }
                            .padding(10)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(SupplierTheme.destructive.opacity(0.1))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                }

                // Commit Button
                Button {
                    PegasusHaptics.heavy()
                    Task { await vm.commitScenario() }
                } label: {
                    HStack {
                        if vm.isCommitting {
                            ProgressView().tint(SupplierTheme.onPrimaryAction)
                            Spacer().frame(width: 8)
                        }
                        Text("mobile_supplier.chrome.optimistic_commit")
                            .font(.headline)
                    }
                    .frame(maxWidth: .infinity)
                    .frame(minHeight: 48)
                    .background(SupplierTheme.primaryAction)
                    .foregroundStyle(SupplierTheme.onPrimaryAction)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                }
                .disabled(vm.isCommitting)

                Spacer(minLength: 40)
            }
            .padding(16)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private var lotRecallTab: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("mobile_supplier.chrome.contaminated_lot")
                        .font(.headline)
                    Text("mobile_supplier.chrome.trace_forward_lineage")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("mobile_supplier.chrome.raw_lot_barcode", text: $vm.lotBarcode)
                        .textFieldStyle(.roundedBorder)
                        .font(.subheadline.monospaced())

                    Button {
                        PegasusHaptics.light()
                        Task { await vm.analyzeLotRecall() }
                    } label: {
                        HStack {
                            if vm.isAnalyzingRecall {
                                ProgressView().tint(SupplierTheme.onPrimaryAction)
                                Spacer().frame(width: 8)
                            }
                            Text("mobile_supplier.chrome.trace_blast_lineage")
                                .font(.subheadline.bold())
                        }
                        .frame(maxWidth: .infinity)
                        .frame(minHeight: 44)
                        .background(SupplierTheme.primaryAction)
                        .foregroundStyle(SupplierTheme.onPrimaryAction)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                    .disabled(vm.isAnalyzingRecall)
                }
                .padding(14)
                .background(SupplierTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 14))

                if let report = vm.recallReport {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("mobile_supplier.chrome.forward_blast_results")
                            .font(.headline)

                        MetricCard(title: "mobile_supplier.chrome.affected_work_orders", count: report.affectedWorkOrders.count, items: report.affectedWorkOrders)
                        MetricCard(title: "mobile_supplier.chrome.affected_finished_lots", count: report.affectedFinishedLots.count, items: report.affectedFinishedLots)
                        MetricCard(title: "mobile_supplier.chrome.quarantined_sscc", count: report.quarantinedSsccPallets.count, items: report.quarantinedSsccPallets, isWarning: true)
                        MetricCard(title: "mobile_supplier.chrome.quarantined_totes", count: report.quarantinedWarehouseTotes.count, items: report.quarantinedWarehouseTotes)
                        MetricCard(title: "mobile_supplier.chrome.in_transit_drop_blocks", count: report.inTransitDriverDropBlocks.count, items: report.inTransitDriverDropBlocks, isWarning: true)
                        MetricCard(title: "mobile_supplier.chrome.impacted_soliq", count: report.impactedSoliqFacturas.count, items: report.impactedSoliqFacturas)

                        Button {
                            PegasusHaptics.heavy()
                            Task { await vm.executeQuarantineBlastRadius() }
                        } label: {
                            HStack {
                                if vm.isQuarantining {
                                    ProgressView().tint(SupplierTheme.onPrimaryAction)
                                    Spacer().frame(width: 8)
                                }
                                Text(LocalizedStringKey(vm.quarantineSuccess ? "mobile_supplier.chrome.quarantine_enforced" : "mobile_supplier.chrome.lockdown_quarantine"))
                                    .font(.headline)
                            }
                            .frame(maxWidth: .infinity)
                            .frame(minHeight: 50)
                            .background(SupplierTheme.destructive)
                            .foregroundStyle(SupplierTheme.onPrimaryAction)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }
                        .disabled(vm.isQuarantining || vm.quarantineSuccess)
                    }
                }

                Spacer(minLength: 40)
            }
            .padding(16)
        }
        .scrollDismissesKeyboard(.interactively)
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
                Text(LocalizedStringKey(title))
                    .font(.subheadline.weight(.semibold))
                Spacer()
                Text("\(count)")
                    .font(.subheadline.bold())
                    .foregroundStyle(isWarning ? SupplierTheme.destructive : SupplierTheme.label)
            }
            if !items.isEmpty {
                Text(items.joined(separator: ", "))
                    .font(.caption.monospaced())
                    .foregroundStyle(SupplierTheme.secondaryLabel)
                    .lineLimit(2)
            }
        }
        .padding(10)
        .background(isWarning ? SupplierTheme.destructive.opacity(0.08) : SupplierTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }
}
