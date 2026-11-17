import SwiftUI

struct ColumnMapperSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Bindable var vm: MigrationViewModel

    @State private var showSampleRows = false
    @State private var pickerProposalColumn: String?

    var body: some View {
        NavigationStack {
            Group {
                if vm.mappingLoading && vm.proposals.isEmpty {
                    SupplierLoadingView(title: "supplier_portal.migrations.mapper_title")
                } else {
                    List {
                        statusHeaderSection
                        actionsSection
                        if showSampleRows && !vm.sampleRows.isEmpty {
                            samplePreviewSection
                        }
                        columnsSection
                    }
                }
            }
            .navigationTitle(LocalizedStringKey("supplier_portal.migrations.mapper_title"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task { await vm.confirmMappings() }
                    } label: {
                        Text(LocalizedStringKey("supplier_portal.migrations.confirm_mapping"))
                    }
                    .disabled(vm.mappingSaving || vm.proposals.isEmpty)
                }
            }
            .sheet(isPresented: Binding(
                get: { pickerProposalColumn != nil },
                set: { if !$0 { pickerProposalColumn = nil } }
            )) {
                if let col = pickerProposalColumn {
                    let current = vm.proposals.first { $0.sourceColumn == col }?.targetField ?? ""
                    SearchableTargetFieldSheet(
                        sourceColumn: col,
                        currentTarget: current,
                        taxonomy: vm.taxonomy
                    ) { selectedTarget in
                        vm.updateProposalTarget(sourceColumn: col, targetField: selectedTarget)
                        pickerProposalColumn = nil
                    }
                }
            }
        }
    }

    private var statusHeaderSection: some View {
        Section {
            if let doc = vm.mappingDoc {
                HStack {
                    Text(LocalizedStringKey(MigrationPack.labelKey(for: doc.pack)))
                        .font(.subheadline.weight(.semibold))
                    Spacer()
                    SupplierStatusBadge(text: "\(doc.aiModel) · \(doc.aiStatus)")
                }

                if let missing = doc.missingRequired, !missing.isEmpty {
                    Text("Missing required: \(missing.joined(separator: ", "))")
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.destructive)
                }

                if let low = doc.lowConfidenceColumns, !low.isEmpty {
                    Text("Low confidence: \(low.joined(separator: ", "))")
                        .font(.caption)
                        .foregroundStyle(SupplierTheme.warning)
                }
            }

            if let error = vm.error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            }
        }
    }

    private var actionsSection: some View {
        Section {
            Button {
                Task { await vm.acceptHighConfidenceMappings() }
            } label: {
                Label("supplier_portal.migrations.accept_high_confidence", systemImage: "wand.and.stars")
            }
            .disabled(vm.mappingSaving || vm.proposals.isEmpty)

            Toggle("supplier_portal.migrations.save_template_toggle", isOn: $vm.saveMappingTemplate)

            Toggle(
                "\(L10n.format("supplier_portal.migrations.sample_values")) (\(vm.sampleRows.count))",
                isOn: $showSampleRows
            )
        }
    }

    private var samplePreviewSection: some View {
        Section(LocalizedStringKey("supplier_portal.migrations.sample_values")) {
            let columns = vm.proposals.map(\.sourceColumn)
            ScrollView(.horizontal, showsIndicators: true) {
                VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                    HStack(spacing: SupplierTheme.spacingSM) {
                        Text("#")
                            .font(.caption2.bold().monospacedDigit())
                            .frame(width: 28, alignment: .leading)
                        ForEach(columns, id: \.self) { col in
                            Text(col)
                                .font(.caption2.bold())
                                .frame(width: 110, alignment: .leading)
                                .lineLimit(1)
                        }
                    }
                    Divider()
                    ForEach(vm.sampleRows) { row in
                        HStack(spacing: SupplierTheme.spacingSM) {
                            Text("\(row.rowIndex)")
                                .font(.caption2.monospacedDigit())
                                .foregroundStyle(SupplierTheme.secondaryLabel)
                                .frame(width: 28, alignment: .leading)
                            ForEach(columns, id: \.self) { col in
                                Text(row.rawData[col] ?? "—")
                                    .font(.caption2)
                                    .frame(width: 110, alignment: .leading)
                                    .lineLimit(1)
                            }
                        }
                    }
                }
                .padding(.vertical, SupplierTheme.spacingXS)
            }
        }
    }

    private var columnsSection: some View {
        Section(LocalizedStringKey("supplier_portal.migrations.source_column")) {
            ForEach(vm.proposals) { proposal in
                Button {
                    pickerProposalColumn = proposal.sourceColumn
                    PegasusHaptics.selection()
                } label: {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                        HStack {
                            Text(proposal.sourceColumn)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(SupplierTheme.label)
                            Spacer()
                            SupplierStatusBadge(
                                text: L10n.format(
                                    "supplier_portal.migrations.confidence_label",
                                    String(Int((proposal.confidence * 100).rounded()))
                                )
                            )
                        }

                        Text(vm.samplePreview(for: proposal.sourceColumn))
                            .font(.caption2)
                            .foregroundStyle(SupplierTheme.secondaryLabel)
                            .lineLimit(2)

                        HStack {
                            Image(systemName: "arrow.right")
                                .font(.caption2)
                                .foregroundStyle(SupplierTheme.tertiaryLabel)
                            Text(targetLabel(for: proposal.targetField))
                                .font(.caption.weight(.medium))
                                .foregroundStyle(proposal.targetField.isEmpty ? SupplierTheme.secondaryLabel : SupplierTheme.primaryAction)
                            Spacer()
                            if proposal.confirmed {
                                Image(systemName: "checkmark.circle.fill")
                                    .foregroundStyle(SupplierTheme.success)
                            }
                        }
                    }
                    .padding(.vertical, SupplierTheme.spacingXS)
                }
                .buttonStyle(.plain)
            }
        }
    }

    private func targetLabel(for key: String) -> String {
        if key.isEmpty {
            return L10n.format("supplier_portal.migrations.ignore_column")
        }
        if let field = vm.taxonomy.first(where: { $0.key == key }) {
            return "\(field.displayLabel) (\(field.key))"
        }
        return key
    }
}

private struct SearchableTargetFieldSheet: View {
    @Environment(\.dismiss) private var dismiss
    let sourceColumn: String
    let currentTarget: String
    let taxonomy: [MigrationCanonicalField]
    let onSelect: (String) -> Void

    @State private var searchText = ""

    private var filteredFields: [MigrationCanonicalField] {
        let q = searchText.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return taxonomy }
        return taxonomy.filter { field in
            field.key.lowercased().contains(q)
                || field.labelEn.lowercased().contains(q)
                || (field.labelRu?.lowercased().contains(q) ?? false)
                || (field.labelUz?.lowercased().contains(q) ?? false)
        }
    }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Button {
                        onSelect("")
                    } label: {
                        HStack {
                            Text(LocalizedStringKey("supplier_portal.migrations.ignore_column"))
                                .foregroundStyle(SupplierTheme.label)
                            Spacer()
                            if currentTarget.isEmpty {
                                Image(systemName: "checkmark")
                                    .foregroundStyle(SupplierTheme.primaryAction)
                            }
                        }
                    }
                }

                Section(LocalizedStringKey("supplier_portal.migrations.target_field")) {
                    ForEach(filteredFields) { field in
                        Button {
                            onSelect(field.key)
                        } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack(spacing: 4) {
                                        Text(field.displayLabel)
                                            .font(.subheadline)
                                            .foregroundStyle(SupplierTheme.label)
                                        if field.required {
                                            Text("*")
                                                .font(.caption.bold())
                                                .foregroundStyle(SupplierTheme.destructive)
                                        }
                                    }
                                    Text("\(field.key) · \(field.type)")
                                        .font(.caption2.monospaced())
                                        .foregroundStyle(SupplierTheme.secondaryLabel)
                                }
                                Spacer()
                                if currentTarget == field.key {
                                    Image(systemName: "checkmark")
                                        .foregroundStyle(SupplierTheme.primaryAction)
                                }
                            }
                        }
                    }
                }
            }
            .searchable(text: $searchText, prompt: Text(LocalizedStringKey("supplier_portal.migrations.target_field")))
            .navigationTitle(sourceColumn)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
            }
        }
    }
}
