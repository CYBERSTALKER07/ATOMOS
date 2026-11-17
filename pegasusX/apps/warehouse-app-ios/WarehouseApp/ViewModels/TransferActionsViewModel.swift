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
    typealias WaivePickShorts = @MainActor (String) async throws -> PickWave
    typealias CreateCycleCount = @MainActor (String, String, Int?) async throws -> CycleCount
    typealias ListCycleCounts = @MainActor () async throws -> CycleCountListResponse
    typealias SubmitCycleCount = @MainActor (String, Int, String?) async throws -> CycleCount
    typealias ListInventoryAdjustments = @MainActor () async throws -> InventoryAdjustmentListResponse
    typealias ApproveInventoryAdjustment = @MainActor (String) async throws -> InventoryAdjustmentMutationResponse
    typealias RejectInventoryAdjustment = @MainActor (String) async throws -> InventoryAdjustmentMutationResponse

    var volumeInput = "20"
    var transferIdInput = ""
    var notesInput = ""
    var putawayProduct = ""
    var putawayLocation = ""
    var putawayQty = "1"
    var putawayExpiry = ""
    var putawayLotCode = ""
    var bins: [WarehouseBinLocation] = []
    var lots: [StockLotItem] = []
    var activeManifestTempMissing = false
    var listBinsOp: () async throws -> WarehouseBinListResponse = {
        try await WarehouseService.listBins()
    }
    var listLotsOp: () async throws -> StockLotListResponse = {
        try await WarehouseService.listLots()
    }
    var checkManifestTempOp: (_ manifestId: String) async throws -> ManifestTemperatureResponse = { manifestId in
        try await WarehouseService.manifestTemperature(manifestId: manifestId)
    }
    var pickManifestId = ""
    var pickScan = ""
    var pickScannedQty = 0
    var pickQtyInput = ""
    var activeWave: PickWave?
    var cycleCounts: [CycleCount] = []
    var selectedCycleCountId: String?
    var countedQtyInput = ""
    var cycleReasonCode = ""
    var cycleExpectedInput = ""
    var adjustments: [InventoryAdjustment] = []
    var cycleCountsEnabled = true
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
    private let waivePickShortsOp: WaivePickShorts
    private let createCycleCountOp: CreateCycleCount
    private let listCycleCountsOp: ListCycleCounts
    private let submitCycleCountOp: SubmitCycleCount
    private let listInventoryAdjustmentsOp: ListInventoryAdjustments
    private let approveInventoryAdjustmentOp: ApproveInventoryAdjustment
    private let rejectInventoryAdjustmentOp: RejectInventoryAdjustment

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
        waivePickShorts: WaivePickShorts? = nil,
        createCycleCount: CreateCycleCount? = nil,
        listCycleCounts: ListCycleCounts? = nil,
        submitCycleCount: SubmitCycleCount? = nil,
        listInventoryAdjustments: ListInventoryAdjustments? = nil,
        approveInventoryAdjustment: ApproveInventoryAdjustment? = nil,
        rejectInventoryAdjustment: RejectInventoryAdjustment? = nil
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
        self.waivePickShortsOp = waivePickShorts ?? { waveId in
            try await WarehouseService.waivePickShorts(waveId: waveId)
        }
        self.createCycleCountOp = createCycleCount ?? { locationId, productId, expectedQty in
            try await WarehouseService.createCycleCount(
                locationId: locationId,
                productId: productId,
                expectedQty: expectedQty
            )
        }
        self.listCycleCountsOp = listCycleCounts ?? {
            try await WarehouseService.listCycleCounts()
        }
        self.submitCycleCountOp = submitCycleCount ?? { countId, countedQty, reasonCode in
            try await WarehouseService.submitCycleCount(
                countId: countId,
                countedQty: countedQty,
                reasonCode: reasonCode
            )
        }
        self.listInventoryAdjustmentsOp = listInventoryAdjustments ?? {
            try await WarehouseService.listInventoryAdjustments()
        }
        self.approveInventoryAdjustmentOp = approveInventoryAdjustment ?? { adjustmentId in
            try await WarehouseService.approveInventoryAdjustment(adjustmentId: adjustmentId)
        }
        self.rejectInventoryAdjustmentOp = rejectInventoryAdjustment ?? { adjustmentId in
            try await WarehouseService.rejectInventoryAdjustment(adjustmentId: adjustmentId)
        }
    }

    var hasShortTasks: Bool {
        activeWave?.tasks?.contains(where: { $0.status.uppercased() == "SHORT" }) ?? false
    }

    static func requiredZone(for lot: StockLotItem) -> String? {
        let explicit = (lot.requiredZone ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        if explicit == "FREEZE" || explicit == "CHILL" {
            return explicit
        }
        let hc = (lot.handlingClass ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        if hc == "FROZEN" {
            return "FREEZE"
        }
        if (lot.requiresColdChain ?? false) || (lot.isPerishable ?? false) ||
            hc == "CHILLED" || hc == "COLD" || hc == "PERISHABLE" ||
            lot.coldLotWarning == "cold_lot_migrated_without_live_baseline" {
            return "CHILL"
        }
        return nil
    }

    static func isColdZoneCompatible(_ zone: String?, requiredZone: String) -> Bool {
        let normalized = (zone ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard !normalized.isEmpty else { return false }
        if requiredZone.uppercased() == "FREEZE" {
            return normalized == "FREEZE" || normalized == "FROZEN" || normalized == "FREEZER"
        }
        return normalized == "CHILL" || normalized == "CHILLED" || normalized == "COLD" ||
            normalized == "REEFER" || normalized == "FREEZE" || normalized == "FROZEN"
    }

    static func isStagingLocation(_ locationId: String, bin: WarehouseBinLocation?) -> Bool {
        let lower = locationId.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if lower == "recv-default" || lower.hasPrefix("recv-") {
            return true
        }
        return (bin?.locationType ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased() == "STAGE"
    }

    var coldWarningLots: [StockLotItem] {
        let binById = Dictionary(uniqueKeysWithValues: bins.map { ($0.locationId, $0) })
        return lots.filter { lot in
            guard lot.quantityOnHand > 0 else { return false }
            if lot.coldLotWarning == "cold_lot_migrated_without_live_baseline" {
                return true
            }
            guard let req = Self.requiredZone(for: lot) else { return false }
            let bin = binById[lot.locationId]
            let zone = bin?.zone ?? lot.locationZone
            return Self.isStagingLocation(lot.locationId, bin: bin) ||
                !Self.isColdZoneCompatible(zone, requiredZone: req)
        }
    }

    var stagingPutawayLots: [StockLotItem] {
        let binById = Dictionary(uniqueKeysWithValues: bins.map { ($0.locationId, $0) })
        return lots.filter { lot in
            lot.quantityOnHand > 0 &&
                ((lot.migratedStaging ?? false) || Self.isStagingLocation(lot.locationId, bin: binById[lot.locationId]))
        }
    }

    var hasColdLotOrStagingAlert: Bool {
        !coldWarningLots.isEmpty || !stagingPutawayLots.isEmpty
    }

    var putawayZoneMismatchRequiredZone: String? {
        let pid = putawayProduct.trimmingCharacters(in: .whitespacesAndNewlines)
        let lid = putawayLocation.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !pid.isEmpty, !lid.isEmpty else { return nil }
        guard let matchingLot = lots.first(where: { $0.productId.caseInsensitiveCompare(pid) == .orderedSame }),
              let req = Self.requiredZone(for: matchingLot),
              let targetBin = bins.first(where: { $0.locationId.caseInsensitiveCompare(lid) == .orderedSame }) else {
            return nil
        }
        let isStage = (targetBin.locationType ?? "").uppercased() == "STAGE"
        if isStage || !Self.isColdZoneCompatible(targetBin.zone, requiredZone: req) {
            return req
        }
        return nil
    }

    var activeWavePreSealWarning: Bool {
        guard let wave = activeWave else { return false }
        if (wave.coldLotMigratedWithoutLiveBaseline ?? false) || (wave.coldChainBaselineMissing ?? false) {
            return true
        }
        if activeManifestTempMissing {
            return true
        }
        let flaggedLotIds = Set(coldWarningLots.map(\.lotId) + stagingPutawayLots.map(\.lotId))
        let binById = Dictionary(uniqueKeysWithValues: bins.map { ($0.locationId, $0) })
        return (wave.tasks ?? []).contains { task in
            flaggedLotIds.contains(task.lotId) ||
                Self.isStagingLocation(task.locationId ?? "", bin: task.locationId.flatMap { binById[$0] })
        }
    }

    func stagePutawayLot(_ lot: StockLotItem) {
        putawayProduct = lot.productId
        putawayQty = "\(max(1, lot.quantityOnHand))"
        putawayExpiry = lot.expiryDate ?? ""
        putawayLotCode = lot.lotCode ?? ""
        if let req = Self.requiredZone(for: lot),
           let preferredBin = bins.first(where: { Self.isColdZoneCompatible($0.zone, requiredZone: req) }) {
            putawayLocation = preferredBin.locationId
        }
    }

    func refreshBinsAndLots() async {
        do {
            async let binsResp = listBinsOp()
            async let lotsResp = listLotsOp()
            self.bins = try await binsResp.bins
            self.lots = try await lotsResp.lots
        } catch {
            // Non-blocking enrichment refresh for cold-lot alerts
        }
    }

    private func refreshManifestTempReadiness(for wave: PickWave?) async {
        guard let wave, !wave.manifestId.isEmpty else {
            activeManifestTempMissing = false
            return
        }
        let taskLotIds = Set((wave.tasks ?? []).map(\.lotId))
        let taskProdIds = Set((wave.tasks ?? []).map(\.productId))
        let hasColdTask = (wave.coldLotMigratedWithoutLiveBaseline ?? false) ||
            (wave.coldChainBaselineMissing ?? false) ||
            lots.contains {
                (taskLotIds.contains($0.lotId) || taskProdIds.contains($0.productId)) &&
                    Self.requiredZone(for: $0) != nil
            }
        guard hasColdTask else {
            activeManifestTempMissing = false
            return
        }
        do {
            let resp = try await checkManifestTempOp(wave.manifestId)
            activeManifestTempMissing = resp.readings.isEmpty
        } catch {
            activeManifestTempMissing = (wave.coldChainBaselineMissing ?? false)
        }
    }

    var selectedOpenCycleCount: CycleCount? {
        let open = cycleCounts.filter { $0.status == "OPEN" }
        return open.first(where: { $0.countId == selectedCycleCountId }) ?? open.first
    }

    static func variance(for count: CycleCount) -> Int? {
        count.varianceQty ?? count.countedQty.map { $0 - count.expectedQty }
    }

    static func signedDeltaString(_ value: Int) -> String {
        value > 0 ? "+\(value)" : "\(value)"
    }

    var nextPendingTask: PickTask? {
        guard let wave = activeWave else { return nil }
        return wave.tasks?.first(where: { t in
            (t.status == "PENDING" || t.status == "SHORT") && (
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
            await self.refreshBinsAndLots()
        }
    }

    private func syncPickQtyInput() {
        if let task = nextPendingTask {
            let defaultQty = task.status == "SHORT" ? (task.quantityPicked ?? 0) : task.quantityRequested
            pickQtyInput = "\(defaultQty)"
        } else {
            pickQtyInput = ""
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
            let created = try await self.createPickWaveOp(mid)
            self.activeWave = created
            self.pickScannedQty = 0
            self.syncPickQtyInput()
            await self.refreshManifestTempReadiness(for: created)
        }
    }

    func loadPickWaves() async {
        let manifest = pickManifestId.isEmpty ? nil : pickManifestId
        await runAction(label: "Load waves") {
            let list = try await self.listPickWavesOp(manifest)
            if let first = list.waves.first {
                let loaded = try await self.getPickWaveOp(first.waveId)
                self.activeWave = loaded
                self.pickScannedQty = 0
                self.syncPickQtyInput()
                await self.refreshManifestTempReadiness(for: loaded)
            } else {
                self.statusMessage = "No pick waves"
                self.activeManifestTempMissing = false
            }
        }
    }

    func confirmCurrentPick(wave: PickWave, task: PickTask) async {
        let trimmed = pickQtyInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            statusMessage = "Picked qty required"
            PegasusHaptics.warning()
            return
        }
        guard let qty = Int(trimmed), qty >= 0, qty <= task.quantityRequested else {
            statusMessage = "Picked qty must be 0...\(task.quantityRequested)"
            PegasusHaptics.warning()
            return
        }
        await confirmPick(wave: wave, task: task, quantity: qty)
    }

    func reportShortZero(wave: PickWave, task: PickTask) async {
        await confirmPick(wave: wave, task: task, quantity: 0)
    }

    func confirmPick(wave: PickWave, task: PickTask, quantity: Int) async {
        guard quantity >= 0, quantity <= task.quantityRequested else {
            statusMessage = "Picked qty must be 0...\(task.quantityRequested)"
            PegasusHaptics.warning()
            return
        }
        await runAction(label: "Confirm pick") {
            let updated = try await self.confirmPickTaskOp(wave.waveId, task.taskId, quantity)
            self.activeWave = updated
            self.pickScan = ""
            self.pickScannedQty = 0
            self.syncPickQtyInput()
            if updated.status == "READY_TO_SEAL" {
                self.statusMessage = "Wave ready to seal"
            } else if quantity < task.quantityRequested {
                self.statusMessage = "Short pick recorded (\(quantity)/\(task.quantityRequested))"
            } else {
                self.statusMessage = "Confirmed · \(updated.status)"
            }
        }
    }

    func waivePickShorts() async {
        guard let wave = activeWave else {
            statusMessage = "No active pick wave"
            PegasusHaptics.warning()
            return
        }
        await runAction(label: "Waive shorts") {
            let updated = try await self.waivePickShortsOp(wave.waveId)
            self.activeWave = updated
            self.pickScannedQty = 0
            self.syncPickQtyInput()
            self.statusMessage = updated.status == "READY_TO_SEAL"
                ? "Shorts waived · wave ready to seal"
                : "Shorts waived · \(updated.status)"
        }
    }

    func selectCycleCount(_ count: CycleCount) {
        selectedCycleCountId = count.countId
        countedQtyInput = "\(count.expectedQty)"
        cycleReasonCode = ""
    }

    private func loadCycleCountsState() async throws {
        let countsResp = try await self.listCycleCountsOp()
        let enabled = countsResp.cycleCountsEnabled ?? true
        self.cycleCounts = countsResp.counts
        self.cycleCountsEnabled = enabled

        if enabled {
            let adjResp = try await self.listInventoryAdjustmentsOp()
            self.adjustments = adjResp.adjustments
        } else {
            self.adjustments = []
        }

        let openCounts = countsResp.counts.filter { $0.status == "OPEN" }
        let prevStillOpen = openCounts.first(where: { $0.countId == self.selectedCycleCountId })
        let resolved = prevStillOpen ?? openCounts.first
        self.selectedCycleCountId = resolved?.countId
        if let resolved {
            if prevStillOpen == nil || self.countedQtyInput.isEmpty {
                self.countedQtyInput = "\(resolved.expectedQty)"
            }
        } else {
            self.countedQtyInput = ""
        }
    }

    func refreshCycleCountsAndAdjustments() async {
        await runAction(label: "Refresh counts") {
            try await self.loadCycleCountsState()
            self.statusMessage = "Loaded \(self.cycleCounts.count) counts"
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
        let expTrimmed = cycleExpectedInput.trimmingCharacters(in: .whitespacesAndNewlines)
        let expectedQty: Int?
        if expTrimmed.isEmpty {
            expectedQty = nil
        } else if let parsed = Int(expTrimmed), parsed >= 0 {
            expectedQty = parsed
        } else {
            statusMessage = "Expected qty must be >= 0"
            PegasusHaptics.warning()
            return
        }
        await runAction(label: "Create count") {
            let created = try await self.createCycleCountOp(lid, pid, expectedQty)
            self.selectedCycleCountId = created.countId
            self.countedQtyInput = "\(created.expectedQty)"
            self.cycleReasonCode = ""
            self.cycleExpectedInput = ""
            try await self.loadCycleCountsState()
            self.statusMessage = "Created count \(created.countId.prefix(8))…"
        }
    }

    func submitCycleCount() async {
        guard let open = selectedOpenCycleCount else {
            statusMessage = "No OPEN counts"
            PegasusHaptics.warning()
            return
        }
        let trimmed = countedQtyInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            statusMessage = "Counted qty required"
            PegasusHaptics.warning()
            return
        }
        guard let countedQty = Int(trimmed), countedQty >= 0 else {
            statusMessage = "Counted qty must be >= 0"
            PegasusHaptics.warning()
            return
        }
        let reasonTrimmed = cycleReasonCode.trimmingCharacters(in: .whitespacesAndNewlines)
        let reason = reasonTrimmed.isEmpty ? nil : reasonTrimmed
        await runAction(label: "Submit count") {
            let submitted = try await self.submitCycleCountOp(open.countId, countedQty, reason)
            self.selectedCycleCountId = nil
            self.countedQtyInput = ""
            self.cycleReasonCode = ""
            try await self.loadCycleCountsState()
            self.statusMessage = "Submitted · \(submitted.status)"
        }
    }

    func approveAdjustment(_ adjustmentId: String) async {
        await runAction(label: "Approve adjustment") {
            _ = try await self.approveInventoryAdjustmentOp(adjustmentId)
            try await self.loadCycleCountsState()
            self.statusMessage = "Adjustment approved"
        }
    }

    func rejectAdjustment(_ adjustmentId: String) async {
        await runAction(label: "Reject adjustment") {
            _ = try await self.rejectInventoryAdjustmentOp(adjustmentId)
            try await self.loadCycleCountsState()
            self.statusMessage = "Adjustment rejected"
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
                  (t.status == "PENDING" || t.status == "SHORT") && (
                      t.productId.caseInsensitiveCompare(trimmed) == .orderedSame ||
                      t.lotId.caseInsensitiveCompare(trimmed) == .orderedSame
                  )
              }) else { return }
        let next = pickScannedQty + 1
        pickScannedQty = next
        pickQtyInput = "\(min(next, task.quantityRequested))"
        if next >= task.quantityRequested {
            await runAction(label: "Confirm pick") {
                self.activeWave = try await self.confirmPickTaskOp(
                    wave.waveId,
                    task.taskId,
                    task.quantityRequested
                )
                self.pickScan = ""
                self.pickScannedQty = 0
                self.syncPickQtyInput()
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
