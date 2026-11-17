import SwiftUI

struct DryRunAndCutoverView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @Bindable var vm: MigrationViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
            dryRunCard
            applyAndRollbackCard
            openOrdersCard
            cutoverCard
        }
    }

    private var dryRunCard: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            SupplierSectionHeader(title: "supplier_portal.migrations.dry_run_title")

            Picker("Apply Mode", selection: $vm.applyMode) {
                Text("MERGE").tag("MERGE")
                Text("SET_ON_HAND").tag("SET_ON_HAND")
            }
            .pickerStyle(.segmented)

            if vm.applyMode == "SET_ON_HAND" {
                Toggle("supplier_portal.migrations.set_on_hand_ack", isOn: $vm.setOnHandConfirmed)
                    .font(.caption)
            }

            HStack(spacing: SupplierTheme.spacingSM) {
                Button {
                    Task { await vm.runDryRun(sessionOnly: true) }
                } label: {
                    Text(LocalizedStringKey("supplier_portal.migrations.run_dry_run"))
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(SupplierTheme.primaryAction)
                .foregroundStyle(SupplierTheme.onPrimaryAction)
                .disabled(vm.busy || vm.selectedSession == nil)

                Button {
                    Task { await vm.runDryRun(sessionOnly: false) }
                } label: {
                    Text("Project Dry-Run")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.bordered)
                .disabled(vm.busy || vm.selectedProjectId == nil)
            }

            if let dryRun = vm.dryRunResult {
                dryRunSummaryView(dryRun)
            }
        }
        .supplierCard()
    }

    @ViewBuilder
    private func dryRunSummaryView(_ dryRun: MigrationDryRunResponse) -> some View {
        let report = dryRun.report
        let summary = dryRun.summary

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            HStack {
                metricCell(label: "Valid", value: "\(report.rowsValid)/\(report.rowsTotal)")
                metricCell(label: "Create", value: "\(report.wouldCreate)")
                metricCell(label: "Update", value: "\(report.wouldUpdate)")
                metricCell(label: L10n.format("supplier_portal.migrations.errors_count"), value: "\(report.rowsInvalid)")
            }

            HStack {
                Text(LocalizedStringKey("supplier_portal.migrations.nasiya_total"))
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
                Spacer()
                Text(MoneyFormat.minor(report.moneyImpactMinor, currency: report.currency))
                    .font(.subheadline.bold().monospacedDigit())
            }

            HStack(spacing: SupplierTheme.spacingSM) {
                Image(systemName: summary.balancedJournalCheck ? "checkmark.seal.fill" : "exclamationmark.octagon.fill")
                    .foregroundStyle(summary.balancedJournalCheck ? SupplierTheme.success : SupplierTheme.destructive)
                Text(LocalizedStringKey(
                    summary.balancedJournalCheck
                        ? "supplier_portal.migrations.balance_status_balanced"
                        : "supplier_portal.migrations.balance_status_unbalanced"
                ))
                .font(.caption.weight(.semibold))
                Spacer()
                Text("Dr \(summary.journalDebitMinor) / Cr \(summary.journalCreditMinor)")
                    .font(.caption2.monospacedDigit())
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
            .padding(SupplierTheme.spacingSM)
            .background(
                (summary.balancedJournalCheck ? SupplierTheme.success : SupplierTheme.destructive).opacity(0.1),
                in: RoundedRectangle(cornerRadius: SupplierTheme.radiusSM)
            )

            if summary.coldLotWarningCount > 0 {
                HStack(spacing: SupplierTheme.spacingSM) {
                    Image(systemName: "thermometer.snowflake")
                        .foregroundStyle(SupplierTheme.warning)
                    Text("\(L10n.format("warehouse_portal.cold_lot.warning_banner")) (\(summary.coldLotWarningCount))")
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.label)
                }
                .padding(SupplierTheme.spacingSM)
                .background(SupplierTheme.warning.opacity(0.14), in: RoundedRectangle(cornerRadius: SupplierTheme.radiusSM))
            }

            if let conflicts = summary.conflicts, !conflicts.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    ForEach(conflicts.prefix(15)) { conflict in
                        HStack(alignment: .top, spacing: 6) {
                            SupplierStatusBadge(text: conflict.severity)
                            VStack(alignment: .leading, spacing: 1) {
                                Text("Row \(conflict.rowIndex) · \(conflict.code)")
                                    .font(.caption2.bold())
                                Text(conflict.message)
                                    .font(.caption2)
                                    .foregroundStyle(SupplierTheme.secondaryLabel)
                            }
                        }
                    }
                }
            }

            HStack(spacing: SupplierTheme.spacingSM) {
                Button {
                    Task { await vm.buildQuarantinedCSV() }
                } label: {
                    Label("supplier_portal.migrations.download_errors_csv", systemImage: "doc.text")
                        .font(.caption)
                }
                .buttonStyle(.bordered)
                .disabled(vm.selectedSession == nil)

                if let csv = vm.quarantinedErrorsCSV, !csv.isEmpty {
                    ShareLink(item: csv, preview: SharePreview("quarantined_rows.csv")) {
                        Label("Share CSV", systemImage: "square.and.arrow.up")
                            .font(.caption)
                    }
                    .buttonStyle(.borderedProminent)
                }
            }
        }
    }

    private func metricCell(label: String, value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(SupplierTheme.secondaryLabel)
            Text(value)
                .font(.subheadline.bold().monospacedDigit())
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private var applyAndRollbackCard: some View {
        if let session = vm.selectedSession {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
                HStack {
                    Text(LocalizedStringKey(MigrationPack.labelKey(for: session.pack ?? "")))
                        .font(.headline)
                    Spacer()
                    SupplierStatusBadge(text: session.status)
                }

                if let liveProgress = realtimeHub.migrationProgressBySession[session.sessionId] {
                    VStack(alignment: .leading, spacing: 4) {
                        ProgressView(
                            value: Double(liveProgress.processedRows),
                            total: Double(max(liveProgress.totalRows, 1))
                        )
                        Text(L10n.format(
                            "supplier_portal.migrations.apply_progress",
                            String(liveProgress.processedRows),
                            String(liveProgress.totalRows)
                        ))
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(SupplierTheme.secondaryLabel)
                    }
                }

                HStack(spacing: SupplierTheme.spacingSM) {
                    Button {
                        Task { await vm.approveSession() }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.migrations.approve_button"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(vm.busy || (vm.applyMode == "SET_ON_HAND" && !vm.setOnHandConfirmed))

                    Button {
                        Task { await vm.applySession() }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.migrations.apply_button"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(SupplierTheme.primaryAction)
                    .foregroundStyle(SupplierTheme.onPrimaryAction)
                    .disabled(vm.busy)
                }

                if let res = vm.applyResult {
                    Text("Created \(res.created) · Updated \(res.updated) · Skipped \(res.skipped) · Chunks \(res.chunksApplied)")
                        .font(.caption.monospacedDigit())
                        .foregroundStyle(SupplierTheme.secondaryLabel)
                }

                if let rollbackUntil = session.rollbackUntil, !rollbackUntil.isEmpty {
                    Text(L10n.format("supplier_portal.migrations.rollback_window_hours", "72"))
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.secondaryLabel)

                    Button(role: .destructive) {
                        Task { await vm.rollbackSession() }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.migrations.rollback_button"))
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .disabled(vm.busy)
                }

                if let rb = vm.rollbackResult {
                    Text("Reverted \(rb.revertedSnapshots) snapshots · Reversed \(rb.reversedInvoices) invoices · Superseded \(rb.supersededOrders) orders")
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.warning)
                }
            }
            .supplierCard()
        }
    }

    private var openOrdersCard: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            SupplierSectionHeader(title: "supplier_portal.migrations.open_orders_title")

            TextField("Order IDs (comma or space separated)", text: $vm.openOrderIdsText, axis: .vertical)
                .textFieldStyle(.roundedBorder)
                .lineLimit(2...4)

            Picker("Target Status", selection: $vm.releaseTargetStatus) {
                Text("PENDING").tag("PENDING")
                Text("CONFIRMED").tag("CONFIRMED")
            }
            .pickerStyle(.segmented)

            Button {
                Task { await vm.releaseOpenOrders() }
            } label: {
                Text(LocalizedStringKey("supplier_portal.migrations.release_orders_button"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .tint(SupplierTheme.primaryAction)
            .foregroundStyle(SupplierTheme.onPrimaryAction)
            .disabled(vm.busy || vm.openOrderIdsText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

            if let rel = vm.releaseResult {
                Text("Released \(rel.releasedOrderIds.count) orders" + (rel.skippedOrderIds.map { " · Skipped \($0.count)" } ?? ""))
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
        }
        .supplierCard()
    }

    private var cutoverCard: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            SupplierSectionHeader(title: "supplier_portal.migrations.cutover_title")

            if let cutover = vm.cutoverResult {
                ForEach(cutover.checklist) { item in
                    HStack(spacing: SupplierTheme.spacingSM) {
                        Image(systemName: item.passed ? "checkmark.circle.fill" : "xmark.circle.fill")
                            .foregroundStyle(item.passed ? SupplierTheme.success : SupplierTheme.destructive)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(item.name)
                                .font(.caption.weight(.semibold))
                            Text(item.detail)
                                .font(.caption2)
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                        }
                    }
                }
            }

            Button {
                Task { await vm.cutoverProject() }
            } label: {
                Text(LocalizedStringKey("supplier_portal.migrations.cutover_button"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .tint(SupplierTheme.primaryAction)
            .foregroundStyle(SupplierTheme.onPrimaryAction)
            .disabled(vm.busy || vm.selectedProjectId == nil)
        }
        .supplierCard()
    }
}
