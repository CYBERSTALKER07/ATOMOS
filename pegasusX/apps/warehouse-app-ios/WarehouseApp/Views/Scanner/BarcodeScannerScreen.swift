//
//  BarcodeScannerScreen.swift
//  WarehouseAppIOS
//
//  Production-grade warehouse optical camera (VisionKit / AVFoundation)
//  and hardware Bluetooth/HID wedge barcode scanner.
//

import SwiftUI
import UIKit

struct BarcodeScannerScreen: View {
    @State private var vm = BarcodeScannerViewModel()
    @FocusState private var isManualFieldFocused: Bool

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                cameraSection
                    .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
                    .padding(.top, PegasusMonochromeTheme.spacingSM)

                statusIndicatorSection
                    .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
                    .padding(.vertical, PegasusMonochromeTheme.spacingSM)

                manualInputSection
                    .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
                    .padding(.bottom, PegasusMonochromeTheme.spacingSM)

                Divider()
                    .background(PegasusMonochromeTheme.separator)

                historyQueueSection
            }
            .background(PegasusMonochromeTheme.background)
            .navigationTitle("Scanner")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        PegasusHaptics.light()
                        vm.clearHistory()
                    } label: {
                        Image(systemName: "trash")
                            .font(.subheadline)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .disabled(vm.scannedRecords.isEmpty)
                    .accessibilityLabel("Clear scan history")
                }
            }
        }
    }

    // MARK: - Camera / Viewfinder Section
    private var cameraSection: some View {
        VStack(spacing: PegasusMonochromeTheme.spacingXS) {
            EANBarcodeScannerView(
                onBarcode: { barcode in
                    vm.handleBarcodeReceived(barcode)
                },
                enabled: vm.scannerActive,
                previewHeight: 220,
                showTorchToggle: true
            )
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(
                        scanBorderColor,
                        lineWidth: 2
                    )
            )
        }
    }

    private var scanBorderColor: Color {
        switch vm.scanState {
        case .idle:
            return PegasusMonochromeTheme.cardBorder
        case .success:
            return PegasusMonochromeTheme.success
        case .error:
            return PegasusMonochromeTheme.destructive
        }
    }

    // MARK: - Status Feedback
    private var statusIndicatorSection: some View {
        HStack(spacing: PegasusMonochromeTheme.spacingSM) {
            switch vm.scanState {
            case .idle:
                Image(systemName: "barcode.viewfinder")
                    .font(.system(.subheadline, weight: .semibold))
                    .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                Text("Align barcode in viewfinder or trigger hardware laser")
                    .font(.caption)
                    .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
            case .success(let code):
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(.subheadline, weight: .bold))
                    .foregroundStyle(PegasusMonochromeTheme.success)
                Text("Scanned: \(code)")
                    .font(.system(.caption, design: .monospaced, weight: .bold))
                    .foregroundStyle(PegasusMonochromeTheme.label)
            case .error(let message):
                Image(systemName: "xmark.octagon.fill")
                    .font(.system(.subheadline, weight: .bold))
                    .foregroundStyle(PegasusMonochromeTheme.destructive)
                Text(message)
                    .font(.caption.bold())
                    .foregroundStyle(PegasusMonochromeTheme.destructive)
            }
            Spacer()
        }
        .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
        .padding(.vertical, PegasusMonochromeTheme.spacingSM)
        .background(PegasusMonochromeTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(PegasusMonochromeTheme.cardBorder, lineWidth: 1)
        )
        .accessibilityElement(children: .combine)
    }

    // MARK: - Manual Barcode Input
    private var manualInputSection: some View {
        HStack(spacing: PegasusMonochromeTheme.spacingSM) {
            TextField("Manual SKU / Bin / Seal barcode…", text: $vm.manualBarcode)
                .textFieldStyle(.plain)
                .font(.system(.subheadline, design: .monospaced))
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .focused($isManualFieldFocused)
                .submitLabel(.done)
                .onSubmit {
                    if vm.submitManualBarcode() {
                        isManualFieldFocused = false
                    }
                }
                .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
                .padding(.vertical, 10)
                .frame(minHeight: 44)
                .background(PegasusMonochromeTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .stroke(PegasusMonochromeTheme.separator, lineWidth: 1)
                )

            Button {
                if vm.submitManualBarcode() {
                    isManualFieldFocused = false
                }
            } label: {
                Text("Accept")
                    .font(.subheadline.bold())
                    .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                    .padding(.horizontal, PegasusMonochromeTheme.spacingMD)
                    .padding(.vertical, 10)
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                    .background(PegasusMonochromeTheme.primaryAction)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
            .buttonStyle(.plain)
            .disabled(vm.isManualSubmitDisabled)
            .accessibilityLabel("Accept manual barcode")
        }
    }

    // MARK: - Scan History List
    private var historyQueueSection: some View {
        Group {
            if vm.scannedRecords.isEmpty {
                VStack(spacing: PegasusMonochromeTheme.spacingSM) {
                    Spacer()
                    Image(systemName: "barcode")
                        .font(.largeTitle)
                        .foregroundStyle(PegasusMonochromeTheme.tertiaryLabel)
                    Text("No Barcodes Scanned")
                        .font(.subheadline.bold())
                        .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                    Text("Scanned bins, pallets, and parcel codes will appear here in chronological order.")
                        .font(.caption)
                        .foregroundStyle(PegasusMonochromeTheme.tertiaryLabel)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 32)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List {
                    Section {
                        ForEach(vm.scannedRecords) { record in
                            HStack(spacing: PegasusMonochromeTheme.spacingMD) {
                                Image(systemName: "checkmark.seal.fill")
                                    .font(.callout)
                                    .foregroundStyle(PegasusMonochromeTheme.label)
                                    .frame(width: 32, height: 32)
                                    .background(PegasusMonochromeTheme.iconTile)
                                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))

                                VStack(alignment: .leading, spacing: 2) {
                                    Text(record.code)
                                        .font(.system(.body, design: .monospaced, weight: .bold))
                                        .foregroundStyle(PegasusMonochromeTheme.label)
                                    Text(record.timestamp.formatted(date: .omitted, time: .standard))
                                        .font(.caption2)
                                        .foregroundStyle(PegasusMonochromeTheme.tertiaryLabel)
                                }
                                .accessibilityElement(children: .combine)

                                Spacer()

                                Button {
                                    UIPasteboard.general.string = record.code
                                    PegasusHaptics.light()
                                } label: {
                                    Image(systemName: "doc.on.doc")
                                        .font(.subheadline)
                                        .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                                        .frame(minWidth: 44, minHeight: 44)
                                        .contentShape(Rectangle())
                                }
                                .buttonStyle(.plain)
                                .accessibilityLabel("Copy barcode \(record.code)")
                            }
                            .padding(.vertical, 4)
                            .listRowBackground(PegasusMonochromeTheme.card)
                        }
                        .onDelete(perform: vm.deleteRecords)
                    } header: {
                        HStack {
                            Text("RECENT SCANS (\(vm.scannedRecords.count))")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                            Spacer()
                        }
                    }
                }
                .listStyle(.insetGrouped)
                .scrollDismissesKeyboard(.interactively)
                .scrollContentBackground(.hidden)
            }
        }
    }
}
