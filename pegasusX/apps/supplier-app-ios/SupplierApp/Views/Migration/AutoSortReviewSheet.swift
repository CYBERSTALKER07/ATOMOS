import SwiftUI

struct AutoSortReviewSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Bindable var vm: MigrationViewModel
    let bundleTree: MigrationBundleTree

    var body: some View {
        NavigationStack {
            Form {
                if let summary = bundleTree.bundle.sortSummary {
                    if summary.conflictingInventoryMode {
                        Section {
                            Text(LocalizedStringKey("supplier_portal.migrations.stock_vs_lots_warning"))
                                .font(.caption)
                                .foregroundStyle(SupplierTheme.warning)

                            Picker("Inventory Mode", selection: $vm.resolvedInventoryMode) {
                                Text(LocalizedStringKey("supplier_portal.migrations.pack_lots")).tag("lots")
                                Text(LocalizedStringKey("supplier_portal.migrations.pack_stock")).tag("stock")
                            }
                            .pickerStyle(.segmented)
                        }
                    }

                    ForEach(summary.streams) { stream in
                        streamSection(stream)
                    }
                } else {
                    Section {
                        Text(LocalizedStringKey("supplier_portal.migrations.no_projects"))
                            .foregroundStyle(SupplierTheme.secondaryLabel)
                    }
                }
            }
            .navigationTitle(LocalizedStringKey("supplier_portal.migrations.sorter_title"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task { await vm.saveSortOverride(bundleTree: bundleTree) }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.migrations.apply_sort_override"))
                    }
                    .disabled(vm.busy)
                }
            }
            .onAppear {
                vm.syncSortDrafts(from: bundleTree.bundle.sortSummary)
            }
        }
    }

    private func streamSection(_ stream: MigrationSortedStreamDecision) -> some View {
        let key = stream.id
        let packBinding = Binding<String>(
            get: { vm.packOverrides[key] ?? stream.detectedPacks.first ?? "" },
            set: { vm.packOverrides[key] = $0 }
        )
        let encodingBinding = Binding<String>(
            get: { vm.encodingOverrides[key] ?? (stream.encoding.isEmpty ? "utf-8" : stream.encoding) },
            set: { vm.encodingOverrides[key] = $0 }
        )
        let ignoreBinding = Binding<Bool>(
            get: { vm.ignoredStreams[key] ?? (stream.ignored ?? false) },
            set: { vm.ignoredStreams[key] = $0 }
        )

        return Section {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                HStack {
                    Text(stream.fileName)
                        .font(.subheadline.weight(.semibold))
                    Spacer()
                    SupplierStatusBadge(
                        text: L10n.format(
                            "supplier_portal.migrations.confidence_label",
                            String(Int((stream.confidence * 100).rounded()))
                        )
                    )
                }

                Text("\(stream.streamKey) • \(stream.totalRows) rows")
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)

                if !stream.headers.isEmpty {
                    Text(stream.headers.prefix(6).joined(separator: ", "))
                        .font(.caption2.monospaced())
                        .foregroundStyle(SupplierTheme.tertiaryLabel)
                        .lineLimit(2)
                }

                if let reason = stream.reviewReason, !reason.isEmpty {
                    Text(reason)
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.warning)
                }
            }

            Picker("supplier_portal.migrations.reassign_stream", selection: packBinding) {
                Text(LocalizedStringKey("supplier_portal.migrations.pack_quarantine")).tag("")
                ForEach(MigrationPack.allCases) { pack in
                    Text(LocalizedStringKey(pack.labelKey)).tag(pack.rawValue)
                }
            }

            Picker("supplier_portal.migrations.encoding_label", selection: encodingBinding) {
                ForEach(MigrationEncodingOption.all, id: \.self) { enc in
                    Text(enc).tag(enc)
                }
            }

            Toggle("supplier_portal.migrations.ignore_column", isOn: ignoreBinding)
        }
    }
}
