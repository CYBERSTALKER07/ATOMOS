import Foundation
import Observation

@Observable
@MainActor
final class TransferActionsViewModel {
    typealias EmergencyTransfer = @MainActor (Double, String?) async throws -> TransferMutationResponse
    typealias ForceReceive = @MainActor (Double, String?) async throws -> TransferMutationResponse
    typealias ReceiveTransfer = @MainActor (String) async throws -> TransferMutationResponse
    typealias CreateBin = @MainActor (String?, String, String) async throws -> WarehouseBinLocation
    typealias PutawayLot = @MainActor (String, String, Int, String?) async throws -> StockLotPutawayResponse
    typealias CreatePickWave = @MainActor (String) async throws -> PickWave
    typealias ListPickWaves = @MainActor (String?) async throws -> PickWaveListResponse
    typealias GetPickWave = @MainActor (String) async throws -> PickWave
    typealias ConfirmPickTask = @MainActor (String, String, Int?) async throws -> PickWave
    typealias CreateCycleCount = @MainActor (String, String) async throws -> CycleCount
    typealias ListCycleCounts = @MainActor () async throws -> CycleCountListResponse
    typealias SubmitCycleCount = @MainActor (String, Int) async throws -> CycleCount

    var volumeInput = "20"
    var transferIdInput = ""
    var notesInput = ""
    var putawayProduct = ""
    var putawayLocation = ""
    var putawayQty = "1"
    var putawayExpiry = ""
    var pickManifestId = ""
    var pickScan = ""
    var pickScannedQty = 0
    var activeWave: PickWave?
    var busy = false
    var statusMessage: String?

    private let emergencyTransferOp: EmergencyTransfer
    private let forceReceiveOp: ForceReceive
    private let receiveTransferOp: ReceiveTransfer
    private let createBinOp: CreateBin
    private let putawayLotOp: PutawayLot
    private let createPickWaveOp: CreatePickWave
    private let listPickWavesOp: ListPickWaves
    private let getPickWaveOp: GetPickWave
    private let confirmPickTaskOp: ConfirmPickTask
    private let createCycleCountOp: CreateCycleCount
    private let listCycleCountsOp: ListCycleCounts
    private let submitCycleCountOp: SubmitCycleCount

    init(
        emergencyTransfer: EmergencyTransfer? = nil,
        forceReceive: ForceReceive? = nil,
        receiveTransfer: ReceiveTransfer? = nil,
        createBin: CreateBin? = nil,
        putawayLot: PutawayLot? = nil,
        createPickWave: CreatePickWave? = nil,
        listPickWaves: ListPickWaves? = nil,
        getPickWave: GetPickWave? = nil,
        confirmPickTask: ConfirmPickTask? = nil,
        createCycleCount: CreateCycleCount? = nil,
        listCycleCounts: ListCycleCounts? = nil,
        submitCycleCount: SubmitCycleCount? = nil
    ) {
        self.emergencyTransferOp = emergencyTransfer ?? { volumeVu, notes in
            try await WarehouseOperationsService.emergencyTransfer(totalVolumeVu: volumeVu, notes: notes)
        }
        self.forceReceiveOp = forceReceive ?? { volumeVu, notes in
            try await WarehouseOperationsService.forceReceive(totalVolumeVu: volumeVu, notes: notes)
        }
        self.receiveTransferOp = receiveTransfer ?? { transferId in
            try await WarehouseOperationsService.receiveTransfer(transferId: transferId)
        }
        self.createBinOp = createBin ?? { locationId, zone, locationType in
            try await WarehouseService.createBin(locationId: locationId, zone: zone, locationType: locationType)
        }
        self.putawayLotOp = putawayLot ?? { productId, locationId, quantity, expiryDate in
            try await WarehouseService.putawayLot(
                productId: productId,
                locationId: locationId,
                quantity: quantity,
                expiryDate: expiryDate
            )
        }
        self.createPickWaveOp = createPickWave ?? { manifestId in
            try await WarehouseService.createPickWave(manifestId: manifestId)
        }
        self.listPickWavesOp = listPickWaves ?? { manifestId in
            try await WarehouseService.listPickWaves(manifestId: manifestId)
        }
        self.getPickWaveOp = getPickWave ?? { waveId in
            try await WarehouseService.getPickWave(waveId: waveId)
        }
        self.confirmPickTaskOp = confirmPickTask ?? { waveId, taskId, quantityPicked in
            try await WarehouseService.confirmPickTask(
                waveId: waveId,
                taskId: taskId,
                quantityPicked: quantityPicked
            )
        }
        self.createCycleCountOp = createCycleCount ?? { locationId, productId in
            try await WarehouseService.createCycleCount(locationId: locationId, productId: productId)
        }
        self.listCycleCountsOp = listCycleCounts ?? {
            try await WarehouseService.listCycleCounts()
        }
        self.submitCycleCountOp = submitCycleCount ?? { countId, countedQty in
            try await WarehouseService.submitCycleCount(countId: countId, countedQty: countedQty)
        }
    }

    var nextPendingTask: PickTask? {
        guard let wave = activeWave else { return nil }
        return wave.tasks?.first(where: { t in
            t.status == "PENDING" && (
                pickScan.isEmpty ||
                t.productId.caseInsensitiveCompare(pickScan) == .orderedSame ||
                t.lotId.caseInsensitiveCompare(pickScan) == .orderedSame
            )
        })
    }

    func handleReconnect() {
        if busy {
            busy = false
            statusMessage = "Connection restored — verify status before retrying."
        }
    }

    func createEmergencyTransfer() async {
        let vol = Double(volumeInput) ?? 20
        let notes = notesInput.isEmpty ? nil : notesInput
        await runMutation(label: "Emergency transfer") {
            try await self.emergencyTransferOp(vol, notes)
        }
    }

    func forceReceivePayload() async {
        let vol = Double(volumeInput) ?? 20
        let notes = notesInput.isEmpty ? nil : notesInput
        await runMutation(label: "Force receive") {
            try await self.forceReceiveOp(vol, notes)
        }
    }

    func receiveInboundTransfer() async {
        let id = transferIdInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else {
            statusMessage = "Transfer ID required"
            PegasusHaptics.warning()
            return
        }
        await runMutation(label: "Receive transfer") {
            try await self.receiveTransferOp(id)
        }
    }

    func ensureStageBin() async {
        let loc = putawayLocation.isEmpty ? nil : putawayLocation
        await runAction(label: "Create bin") {
            _ = try await self.createBinOp(loc, "RECV", "STAGE")
        }
    }

    func performPutawayLot() async {
        let pid = putawayProduct.trimmingCharacters(in: .whitespacesAndNewlines)
        let lid = putawayLocation.trimmingCharacters(in: .whitespacesAndNewlines)
        let qty = Int(putawayQty) ?? 0
        guard !pid.isEmpty, !lid.isEmpty, qty > 0 else {
            statusMessage = "Product, location, qty required"
            PegasusHaptics.warning()
            return
        }
        let expiry = putawayExpiry.isEmpty ? nil : putawayExpiry
        await runAction(label: "Putaway") {
            _ = try await self.putawayLotOp(pid, lid, qty, expiry)
        }
    }

    func createPickWave() async {
        let mid = pickManifestId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !mid.isEmpty else {
            statusMessage = "Manifest ID required"
            PegasusHaptics.warning()
            return
        }
        await runAction(label: "Create wave") {
            self.activeWave = try await self.createPickWaveOp(mid)
            self.pickScannedQty = 0
        }
    }

    func loadPickWaves() async {
        let manifest = pickManifestId.isEmpty ? nil : pickManifestId
        await runAction(label: "Load waves") {
            let list = try await self.listPickWavesOp(manifest)
            if let first = list.waves.first {
                self.activeWave = try await self.getPickWaveOp(first.waveId)
                self.pickScannedQty = 0
            } else {
                self.statusMessage = "No pick waves"
            }
        }
    }

    func confirmCurrentPick(wave: PickWave, task: PickTask) async {
        let qty = pickScannedQty > 0
            ? min(pickScannedQty, task.quantityRequested)
            : task.quantityRequested
        await runAction(label: "Confirm pick") {
            self.activeWave = try await self.confirmPickTaskOp(wave.waveId, task.taskId, qty)
            self.pickScan = ""
            self.pickScannedQty = 0
        }
    }

    func createCycleCount() async {
        let lid = putawayLocation.trimmingCharacters(in: .whitespacesAndNewlines)
        let pid = putawayProduct.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !lid.isEmpty, !pid.isEmpty else {
            statusMessage = "Location + product required"
            PegasusHaptics.warning()
            return
        }
        await runAction(label: "Create count") {
            _ = try await self.createCycleCountOp(lid, pid)
        }
    }

    func submitFirstOpenExpected() async {
        await runAction(label: "Submit count") {
            let list = try await self.listCycleCountsOp()
            guard let open = list.counts.first(where: { $0.status == "OPEN" }) else {
                self.statusMessage = "No OPEN counts"
                return
            }
            _ = try await self.submitCycleCountOp(open.countId, open.expectedQty)
        }
    }

    func onPickBarcode(_ code: String) async {
        let trimmed = code.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !busy else { return }
        pickScan = trimmed
        if putawayProduct.isEmpty {
            putawayProduct = trimmed
        }
        guard let wave = activeWave,
              let task = wave.tasks?.first(where: { t in
                  t.status == "PENDING" && (
                      t.productId.caseInsensitiveCompare(trimmed) == .orderedSame ||
                      t.lotId.caseInsensitiveCompare(trimmed) == .orderedSame
                  )
              }) else { return }
        let next = pickScannedQty + 1
        pickScannedQty = next
        if next >= task.quantityRequested {
            await runAction(label: "Confirm pick") {
                self.activeWave = try await self.confirmPickTaskOp(
                    wave.waveId,
                    task.taskId,
                    task.quantityRequested
                )
                self.pickScan = ""
                self.pickScannedQty = 0
            }
        } else {
            PegasusHaptics.selection()
            statusMessage = "Scanned \(next) / \(task.quantityRequested)"
        }
    }

    private func runMutation(
        label: String,
        _ block: @escaping @MainActor () async throws -> TransferMutationResponse
    ) async {
        busy = true
        statusMessage = nil
        defer { busy = false }
        do {
            let response = try await block()
            statusMessage = "\(label) succeeded · \(response.state)"
            PegasusHaptics.success()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    private func runAction(
        label: String,
        _ block: @escaping @MainActor () async throws -> Void
    ) async {
        busy = true
        statusMessage = nil
        defer { busy = false }
        do {
            try await block()
            if statusMessage == nil {
                statusMessage = "\(label) succeeded"
            }
            PegasusHaptics.success()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
