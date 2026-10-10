import SwiftUI

struct TruckSidebar: View {
    @Bindable var viewModel: HomeViewModel
    @Binding var isExpanded: Bool

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                if isExpanded {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("MANIFEST BOARD")
                            .font(.system(size: 13, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.secondary)
                        Text("DRAFT · LOADING · SEALED · DISPATCHED")
                            .font(.system(size: 9, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.tertiary)
                    }
                }
                Spacer(minLength: 0)
                Button {
                    withAnimation(.spring(response: 0.35, dampingFraction: 0.8)) {
                        isExpanded.toggle()
                    }
                } label: {
                    Image(systemName: "sidebar.left")
                        .foregroundStyle(TermTheme.secondary)
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, isExpanded ? 16 : 12)
            .padding(.top, 8)

            Group {
                if viewModel.loadingTrucks && viewModel.trucks.isEmpty {
                    PayloadLoadingView(
                        title: "LOADING_VEHICLES",
                        message: "Refreshing supplier fleet availability for this shift."
                    )
                } else if viewModel.trucks.isEmpty {
                    PayloadStateView(
                        variant: .truck,
                        title: "NO_VEHICLES",
                        message: "Pull to refresh once dispatch assigns trucks.",
                        tone: .warning
                    )
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if isExpanded {
                    expandedTruckList
                } else {
                    collapsedTruckRail
                }
            }
            if let err = viewModel.error, isExpanded {
                Text(err)
                    .font(.footnote)
                    .foregroundStyle(.red)
                    .padding(.horizontal)
            }
        }
    }

    private var expandedTruckList: some View {
        Group {
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
                        ForEach(viewModel.batchSealFailures.indices, id: \.self) { idx in
                            let row = viewModel.batchSealFailures[idx]
                            VStack(alignment: .leading, spacing: 4) {
                                Text("\(row.manifestId ?? "manifest"): \(row.status ?? "failed")")
                                    .font(.caption.monospaced())
                                if let explain = row.explain {
                                    ExplainStatusBanner(explain: explain)
                                }
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
            List(selection: Binding(
                get: { viewModel.selectedTruckId },
                set: { id in if let id { Task { await viewModel.selectTruck(id) } } }
            )) {
                ForEach(ManifestBoard.group(viewModel.trucks), id: \.state) { column in
                    Section(column.trucks.isEmpty ? "\(column.state) · empty" : "\(column.state) · \(column.trucks.count)") {
                        if column.trucks.isEmpty {
                            Text("No \(column.state.lowercased()) manifests")
                                .font(.caption)
                                .foregroundStyle(TermTheme.tertiary)
                        } else {
                            ForEach(column.trucks) { truck in
                                TruckRow(truck: truck)
                                    .tag(truck.id)
                            }
                        }
                    }
                }
                if !ManifestBoard.unassigned(viewModel.trucks).isEmpty {
                    Section("NO OPEN MANIFEST") {
                        ForEach(ManifestBoard.unassigned(viewModel.trucks)) { truck in
                            TruckRow(truck: truck)
                                .tag(truck.id)
                        }
                    }
                }
            }
            .refreshable { await viewModel.refreshTrucks() }
        }
    }

    private var collapsedTruckRail: some View {
        ScrollView(showsIndicators: false) {
            VStack(spacing: 8) {
                ForEach(viewModel.trucks) { truck in
                    let selected = viewModel.selectedTruckId == truck.id
                    Button {
                        Task { await viewModel.selectTruck(truck.id) }
                    } label: {
                        Image(systemName: "truck.box.fill")
                            .font(.system(size: 20, weight: selected ? .bold : .regular))
                            .foregroundStyle(selected ? TermTheme.accent : TermTheme.secondary)
                            .frame(width: 44, height: 44)
                            .background(
                                RoundedRectangle(cornerRadius: 10, style: .continuous)
                                    .fill(selected ? TermTheme.accent.opacity(0.12) : Color.clear)
                            )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.vertical, 8)
        }
    }
}

struct TruckRow: View {
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
        var parts = [truck.licensePlate, truck.vehicleClass]
            .compactMap { $0?.isEmpty == false ? $0 : nil }
        if let status = truck.truckStatus, ManifestBoard.isBoardState(status) {
            parts.append(status)
        }
        if let max = truck.maxVolumeVu, max > 0 {
            parts.append(String(format: "%.0f/%.0f VU", truck.usedVolumeVu ?? 0, max))
        }
        return parts.joined(separator: " — ")
    }
}
