//
//  BarcodeScannerViewModel.swift
//  WarehouseAppIOS
//

import AudioToolbox
import Foundation
import Observation

struct ScannedBarcodeRecord: Identifiable, Sendable, Hashable {
    let id = UUID()
    let code: String
    let timestamp: Date
}

@Observable
@MainActor
final class BarcodeScannerViewModel {
    enum ScanState: Equatable {
        case idle
        case success(String)
        case error(String)
    }

    var scanState: ScanState = .idle
    var scannedRecords: [ScannedBarcodeRecord] = []
    var manualBarcode: String = ""
    var scannerActive: Bool = true

    private var resetTask: Task<Void, Never>?
    private let resetDelay: Duration
    private let playSounds: Bool

    init(resetDelay: Duration = .seconds(2), playSounds: Bool = true) {
        self.resetDelay = resetDelay
        self.playSounds = playSounds
    }

    var isManualSubmitDisabled: Bool {
        manualBarcode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func clearHistory() {
        resetTask?.cancel()
        scannedRecords.removeAll()
        scanState = .idle
    }

    func submitManualBarcode() -> Bool {
        let trimmed = manualBarcode.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        manualBarcode = ""
        handleBarcodeReceived(trimmed)
        return true
    }

    func handleBarcodeReceived(_ rawCode: String) {
        let code = rawCode.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !code.isEmpty else {
            triggerErrorFeedback(message: "Empty barcode")
            return
        }

        let record = ScannedBarcodeRecord(code: code, timestamp: Date())
        scannedRecords.insert(record, at: 0)

        scanState = .success(code)
        PegasusHaptics.success()
        if playSounds {
            AudioServicesPlaySystemSound(1052)
        }

        resetTask?.cancel()
        let delay = resetDelay
        resetTask = Task { @MainActor [weak self] in
            try? await Task.sleep(for: delay)
            guard !Task.isCancelled, let self else { return }
            if case .success(let active) = self.scanState, active == code {
                self.scanState = .idle
            }
        }
    }

    func deleteRecords(at offsets: IndexSet) {
        scannedRecords.remove(atOffsets: offsets)
    }

    private func triggerErrorFeedback(message: String) {
        scanState = .error(message)
        PegasusHaptics.error()
        if playSounds {
            AudioServicesPlaySystemSound(1053)
        }

        resetTask?.cancel()
        let delay = resetDelay
        resetTask = Task { @MainActor [weak self] in
            try? await Task.sleep(for: delay)
            guard !Task.isCancelled, let self else { return }
            if case .error = self.scanState {
                self.scanState = .idle
            }
        }
    }
}
