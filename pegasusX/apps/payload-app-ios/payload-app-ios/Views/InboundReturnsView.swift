//
//  InboundReturnsView.swift
//  payload-app-ios
//
//  Mirrors payload-terminal inboundReturns.tsx: queue + history, EAN scan,
//  tap-to-select rows, restock / write-off confirm.
//

import SwiftUI

struct InboundReturnsView: View {
    var online: Bool = true
    @Environment(\.dismiss) private var dismiss
    @State private var viewModel = InboundReturnsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        NavigationStack {
            VStack(spacing: 0) {
                scanHeader(vm: vm)
                Picker("View", selection: $vm.tab) {
                    ForEach(InboundReturnsViewModel.Tab.allCases, id: \.self) { t in
                        Text(t.rawValue).tag(t)
                    }
                }
                .pickerStyle(.segmented)
                .onChange(of: vm.tab) { _, _ in
                    PegasusHaptics.selection()
                }
                .padding(.horizontal, TermTheme.s16)
                .padding(.bottom, TermTheme.s8)

                Group {
                    if vm.loading {
                        PayloadLoadingView(title: "Loading inbound returns…")
                    } else if let error = vm.error {
                        PayloadErrorView(message: error) { Task { await vm.load() } }
                    } else if vm.tab == .queue {
                        queueList(vm: vm)
                    } else {
                        historyList(vm: vm)
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("warehouse_portal.returns.text.inbound_returns")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") {
                        PegasusHaptics.light()
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                            .frame(minWidth: 44, minHeight: 44)
                    }
                    .accessibilityLabel("Refresh returns")
                }
            }
            .task { await vm.load() }
            .refreshable { await vm.load() }
        }
    }

    private func scanHeader(vm: InboundReturnsViewModel) -> some View {
        @Bindable var bindableVM = vm
        return VStack(alignment: .leading, spacing: TermTheme.s8) {
            EANBarcodeScannerView(
                onBarcode: { code in
                    PegasusHaptics.light()
                    vm.barcode = code
                    Task { await vm.handleScan(online: online) }
                },
                enabled: vm.scannerEnabled && !vm.scanning
            )
            Text("mobile_payload.ui.ean_barcode_2")
                .font(.caption)
                .foregroundStyle(TermTheme.secondary)
            HStack(spacing: TermTheme.s8) {
                TextField("mobile_payload.ui.scan_or_type_ean", text: $bindableVM.barcode)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .padding(TermTheme.s12)
                    .frame(minHeight: 44)
                    .background(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    .onSubmit { Task { await vm.handleScan(online: online) } }
                Button(vm.scanning ? "…" : "Scan") {
                    PegasusHaptics.light()
                    Task { await vm.handleScan(online: online) }
                }
                .frame(minWidth: 64, minHeight: 44)
                .buttonStyle(.borderedProminent)
                .disabled(vm.scanning || vm.barcode.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            HStack(spacing: TermTheme.s8) {
                Button("mobile_payload.ui.restock") {
                    PegasusHaptics.success()
                    Task { await vm.confirm(disposition: "RESTOCK") }
                }
                .frame(maxWidth: .infinity, minHeight: 44)
                .buttonStyle(.borderedProminent)
                .tint(.green)
                Button("supplier_portal.returns.text.write_off") {
                    PegasusHaptics.warning()
                    Task { await vm.confirm(disposition: "WRITE_OFF") }
                }
                .frame(maxWidth: .infinity, minHeight: 44)
                .buttonStyle(.borderedProminent)
                .tint(.red)
            }
            if vm.queuedScans > 0 {
                Text(L10n.format("mobile_payload.ui.queuedscans_scan_s_queued_offline_2", "\(vm.queuedScans)"))
                    .font(.caption2)
                    .foregroundStyle(.orange)
            }
            if let statusMessage = vm.statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(TermTheme.s16)
        .background(TermTheme.bg)
    }

    private func queueList(vm: InboundReturnsViewModel) -> some View {
        @Bindable var bindableVM = vm
        return Group {
            if vm.rows.isEmpty {
                ContentUnavailableView("No trucks at gate", systemImage: "arrow.uturn.backward.circle")
            } else {
                InboundReturnsList(rows: vm.rows, selectable: true, selected: $bindableVM.selected)
            }
        }
    }

    private func historyList(vm: InboundReturnsViewModel) -> some View {
        @Bindable var bindableVM = vm
        return Group {
            if vm.history.isEmpty {
                ContentUnavailableView("No history", systemImage: "clock.arrow.circlepath")
            } else {
                InboundReturnsList(rows: vm.history, selectable: false, selected: $bindableVM.selected)
            }
        }
    }
}
