package com.pegasusx.warehouse.ui.screens.transfers

import com.pegasusx.warehouse.data.model.CycleCount
import com.pegasusx.warehouse.data.model.InventoryAdjustment
import com.pegasusx.warehouse.data.model.PickTask
import com.pegasusx.warehouse.data.model.PickWave
import com.pegasusx.warehouse.data.model.StockLot
import com.pegasusx.warehouse.data.model.WarehouseBinLocation
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository

data class TransferWmsState(
    val activeWave: PickWave? = null,
    val pickQtyText: String = "",
    val cycleCounts: List<CycleCount> = emptyList(),
    val selectedCountId: String? = null,
    val countedQtyText: String = "",
    val cycleReasonCode: String = "",
    val adjustments: List<InventoryAdjustment> = emptyList(),
    val pickWavesEnabled: Boolean = true,
    val cycleCountsEnabled: Boolean = true,
    val bins: List<WarehouseBinLocation> = emptyList(),
    val lots: List<StockLot> = emptyList(),
    val manifestTempMissing: Boolean = false,
)

object TransferWmsHelpers {
    fun editablePickTasks(wave: PickWave?): List<PickTask> =
        wave?.tasks?.filter { it.status == "PENDING" || it.status == "SHORT" }.orEmpty()

    fun hasShortTasks(wave: PickWave?): Boolean =
        wave?.tasks?.any { it.status.equals("SHORT", ignoreCase = true) } == true

    fun defaultPickQtyText(task: PickTask?): String = when {
        task == null -> ""
        task.status == "SHORT" -> task.quantityPicked.toString()
        else -> task.quantityRequested.toString()
    }

    fun parsePickQty(rawText: String, maxRequested: Long): Result<Long> {
        val trimmed = rawText.trim()
        if (trimmed.isEmpty()) {
            return Result.failure(IllegalArgumentException("Picked quantity required"))
        }
        val parsed = trimmed.toLongOrNull()
            ?: return Result.failure(IllegalArgumentException("Picked quantity must be an integer"))
        if (parsed < 0L || parsed > maxRequested) {
            return Result.failure(IllegalArgumentException("Picked quantity must be 0..$maxRequested"))
        }
        return Result.success(parsed)
    }

    fun parseCountedQty(rawText: String): Result<Long> {
        val trimmed = rawText.trim()
        if (trimmed.isEmpty()) {
            return Result.failure(IllegalArgumentException("Counted quantity required"))
        }
        val parsed = trimmed.toLongOrNull()
            ?: return Result.failure(IllegalArgumentException("Counted quantity must be an integer"))
        if (parsed < 0L) {
            return Result.failure(IllegalArgumentException("Counted quantity must be >= 0"))
        }
        return Result.success(parsed)
    }

    fun computeVariance(count: CycleCount): Long? =
        count.varianceQty ?: count.countedQty?.let { it - count.expectedQty }

    fun formatSignedDelta(delta: Long): String = if (delta > 0L) "+$delta" else delta.toString()

    fun selectedOpenCount(state: TransferWmsState): CycleCount? {
        val openCounts = state.cycleCounts.filter { it.status == "OPEN" }
        return openCounts.firstOrNull { it.countId == state.selectedCountId } ?: openCounts.firstOrNull()
    }

    fun requiredColdZone(lot: StockLot): String? {
        val explicit = lot.requiredZone.orEmpty().trim().uppercase()
        if (explicit == "FREEZE" || explicit == "CHILL") return explicit
        val hc = lot.handlingClass.orEmpty().trim().uppercase()
        if (hc == "FROZEN") return "FREEZE"
        if (
            lot.requiresColdChain ||
            lot.isPerishable ||
            hc == "CHILLED" ||
            hc == "COLD" ||
            hc == "PERISHABLE" ||
            lot.coldLotWarning == "cold_lot_migrated_without_live_baseline"
        ) {
            return "CHILL"
        }
        return null
    }

    fun isColdZoneCompatible(zone: String?, requiredZone: String): Boolean {
        val normalized = zone.orEmpty().trim().uppercase()
        if (normalized.isEmpty()) return false
        return if (requiredZone.uppercase() == "FREEZE") {
            normalized == "FREEZE" || normalized == "FROZEN" || normalized == "FREEZER"
        } else {
            normalized == "CHILL" ||
                normalized == "CHILLED" ||
                normalized == "COLD" ||
                normalized == "REEFER" ||
                normalized == "FREEZE" ||
                normalized == "FROZEN"
        }
    }

    fun isStagingLocation(locationId: String?, bin: WarehouseBinLocation?): Boolean {
        val normalized = locationId.orEmpty().trim().lowercase()
        if (normalized == "recv-default" || normalized.startsWith("recv-")) return true
        return bin?.locationType?.trim()?.uppercase() == "STAGE"
    }

    fun coldWarningLots(lots: List<StockLot>, bins: List<WarehouseBinLocation>): List<StockLot> {
        val binById = bins.associateBy { it.locationId }
        return lots.filter { lot ->
            if (lot.quantityOnHand <= 0L) return@filter false
            if (lot.coldLotWarning == "cold_lot_migrated_without_live_baseline") return@filter true
            val req = requiredColdZone(lot) ?: return@filter false
            val bin = binById[lot.locationId]
            isStagingLocation(lot.locationId, bin) || !isColdZoneCompatible(bin?.zone ?: lot.locationZone, req)
        }
    }

    fun stagingPutawayLots(lots: List<StockLot>, bins: List<WarehouseBinLocation>): List<StockLot> {
        val binById = bins.associateBy { it.locationId }
        return lots.filter { lot ->
            lot.quantityOnHand > 0L && (lot.migratedStaging || isStagingLocation(lot.locationId, binById[lot.locationId]))
        }
    }

    fun putawayZoneMismatch(
        productId: String,
        locationId: String,
        lots: List<StockLot>,
        bins: List<WarehouseBinLocation>,
    ): String? {
        val pid = productId.trim()
        val lid = locationId.trim()
        if (pid.isEmpty() || lid.isEmpty()) return null
        val matchingLot = lots.firstOrNull { it.productId.equals(pid, ignoreCase = true) } ?: return null
        val req = requiredColdZone(matchingLot) ?: return null
        val targetBin = bins.firstOrNull { it.locationId.equals(lid, ignoreCase = true) } ?: return null
        if (targetBin.locationType.equals("STAGE", ignoreCase = true) || !isColdZoneCompatible(targetBin.zone, req)) {
            return req
        }
        return null
    }

    fun recommendedBinForLot(lot: StockLot, bins: List<WarehouseBinLocation>): String? {
        val req = requiredColdZone(lot)
        if (req != null) {
            bins.firstOrNull { it.isActive && isColdZoneCompatible(it.zone, req) }?.let { return it.locationId }
        }
        return bins.firstOrNull {
            it.isActive && !it.locationType.equals("STAGE", ignoreCase = true) && !it.locationType.equals("QUARANTINE", ignoreCase = true)
        }?.locationId
    }

    fun isWavePreSealColdWarning(state: TransferWmsState): Boolean {
        val wave = state.activeWave ?: return false
        if (wave.coldLotMigratedWithoutLiveBaseline || wave.coldChainBaselineMissing || state.manifestTempMissing) {
            return true
        }
        val flaggedLotIds = (coldWarningLots(state.lots, state.bins) + stagingPutawayLots(state.lots, state.bins))
            .map { it.lotId }
            .toSet()
        val binById = state.bins.associateBy { it.locationId }
        return wave.tasks.any { task ->
            flaggedLotIds.contains(task.lotId) || isStagingLocation(task.locationId, binById[task.locationId])
        }
    }
}

class TransferActionsOperationsController(
    private val opsRepository: WarehouseOperationsRepository,
    private val getStateOverride: (() -> TransferWmsState)? = null,
    private val setStateOverride: ((TransferWmsState) -> Unit)? = null,
) {
    private var backingState = TransferWmsState()

    var state: TransferWmsState
        get() = getStateOverride?.invoke() ?: backingState
        private set(value) {
            backingState = value
            setStateOverride?.invoke(value)
        }

    fun updatePickQtyText(value: String) {
        state = state.copy(pickQtyText = value)
    }

    fun updateCountedQtyText(value: String) {
        state = state.copy(countedQtyText = value)
    }

    fun updateCycleReasonCode(value: String) {
        state = state.copy(cycleReasonCode = value)
    }

    fun selectOpenCount(count: CycleCount) {
        state = state.copy(
            selectedCountId = count.countId,
            countedQtyText = count.expectedQty.toString(),
            cycleReasonCode = "",
        )
    }

    suspend fun refreshBinsAndLots(): Result<Unit> {
        val binsRes = runCatching { opsRepository.listBins() }.getOrNull()
        val lotsRes = runCatching { opsRepository.listLots() }.getOrNull()
        val nextBins = if (binsRes?.isSuccessful == true) binsRes.body()?.bins.orEmpty() else state.bins
        val nextLots = if (lotsRes?.isSuccessful == true) lotsRes.body()?.lots.orEmpty() else state.lots
        state = state.copy(bins = nextBins, lots = nextLots)
        return Result.success(Unit)
    }

    suspend fun refreshManifestTempReadiness(manifestId: String) {
        val mid = manifestId.trim()
        if (mid.isEmpty()) {
            state = state.copy(manifestTempMissing = false)
            return
        }
        val res = runCatching { opsRepository.getManifestTemperature(mid) }.getOrNull()
        if (res?.isSuccessful == true) {
            state = state.copy(manifestTempMissing = res.body()?.readings.isNullOrEmpty())
        }
    }

    suspend fun createPickWave(manifestId: String): Result<PickWave> {
        val trimmed = manifestId.trim()
        if (trimmed.isEmpty()) {
            return Result.failure(IllegalArgumentException("Manifest ID required"))
        }
        val res = opsRepository.createPickWave(trimmed)
        val body = res.body()
        if (!res.isSuccessful || body == null) {
            return Result.failure(IllegalStateException("Create wave failed (${res.code()})"))
        }
        val nextTask = TransferWmsHelpers.editablePickTasks(body).firstOrNull()
        state = state.copy(
            activeWave = body,
            pickQtyText = TransferWmsHelpers.defaultPickQtyText(nextTask),
        )
        return Result.success(body)
    }

    suspend fun loadPickWaves(manifestIdFilter: String? = null): Result<PickWave> {
        val res = opsRepository.listPickWaves(manifestIdFilter?.trim()?.ifEmpty { null })
        if (res.code() == 409) {
            state = state.copy(pickWavesEnabled = false, activeWave = null, pickQtyText = "")
            return Result.failure(IllegalStateException("Pick waves disabled"))
        }
        val wave = res.body()?.waves?.firstOrNull()
        if (!res.isSuccessful || wave == null) {
            return Result.failure(IllegalStateException("No pick waves (${res.code()})"))
        }
        val detail = opsRepository.getPickWave(wave.waveId)
        val resolved = detail.body() ?: wave
        val nextTask = TransferWmsHelpers.editablePickTasks(resolved).firstOrNull()
        state = state.copy(
            activeWave = resolved,
            pickWavesEnabled = res.body()?.pickWavesEnabled ?: true,
            pickQtyText = TransferWmsHelpers.defaultPickQtyText(nextTask),
        )
        return Result.success(resolved)
    }

    suspend fun confirmPickWithInput(task: PickTask, rawQtyText: String): Result<PickWave> {
        val parsed = TransferWmsHelpers.parsePickQty(rawQtyText, task.quantityRequested)
        val qty = parsed.getOrElse { return Result.failure(it) }
        return confirmPickExplicit(task, qty)
    }

    suspend fun confirmPickExplicit(task: PickTask, quantityPicked: Long): Result<PickWave> {
        val wave = state.activeWave
            ?: return Result.failure(IllegalStateException("No active pick wave"))
        if (quantityPicked < 0L || quantityPicked > task.quantityRequested) {
            return Result.failure(
                IllegalArgumentException("Picked quantity must be 0..${task.quantityRequested}"),
            )
        }
        val res = opsRepository.confirmPickTask(wave.waveId, task.taskId, quantityPicked)
        val updated = res.body()
        if (!res.isSuccessful || updated == null) {
            return Result.failure(IllegalStateException("Confirm failed (${res.code()})"))
        }
        val nextTask = TransferWmsHelpers.editablePickTasks(updated).firstOrNull()
        state = state.copy(
            activeWave = updated,
            pickQtyText = TransferWmsHelpers.defaultPickQtyText(nextTask),
        )
        return Result.success(updated)
    }

    suspend fun waivePickShorts(): Result<PickWave> {
        val wave = state.activeWave
            ?: return Result.failure(IllegalStateException("No active pick wave"))
        val res = opsRepository.waivePickShorts(wave.waveId)
        val updated = res.body()
        if (!res.isSuccessful || updated == null) {
            return Result.failure(IllegalStateException("Waive shorts failed (${res.code()})"))
        }
        val nextTask = TransferWmsHelpers.editablePickTasks(updated).firstOrNull()
        state = state.copy(
            activeWave = updated,
            pickQtyText = TransferWmsHelpers.defaultPickQtyText(nextTask),
        )
        return Result.success(updated)
    }

    suspend fun refreshCycleCountsAndAdjustments(): Result<Unit> {
        val countsRes = opsRepository.listCycleCounts()
        if (countsRes.code() == 409) {
            state = state.copy(
                cycleCountsEnabled = false,
                cycleCounts = emptyList(),
                adjustments = emptyList(),
                selectedCountId = null,
                countedQtyText = "",
            )
            return Result.success(Unit)
        }
        if (!countsRes.isSuccessful) {
            return Result.failure(IllegalStateException("List cycle counts failed (${countsRes.code()})"))
        }
        val countsBody = countsRes.body()
        val counts = countsBody?.counts.orEmpty()

        val adjRes = opsRepository.listInventoryAdjustments()
        if (!adjRes.isSuccessful && adjRes.code() != 409) {
            return Result.failure(IllegalStateException("List adjustments failed (${adjRes.code()})"))
        }
        val adjustments = adjRes.body()?.adjustments.orEmpty()

        val openCounts = counts.filter { it.status == "OPEN" }
        val prevSelectedStillOpen = openCounts.firstOrNull { it.countId == state.selectedCountId }
        val resolvedSelected = prevSelectedStillOpen ?: openCounts.firstOrNull()
        val resolvedCountedText = when {
            resolvedSelected == null -> ""
            prevSelectedStillOpen != null && state.countedQtyText.isNotEmpty() -> state.countedQtyText
            else -> resolvedSelected.expectedQty.toString()
        }

        state = state.copy(
            cycleCounts = counts,
            adjustments = adjustments,
            cycleCountsEnabled = countsBody?.cycleCountsEnabled ?: true,
            selectedCountId = resolvedSelected?.countId,
            countedQtyText = resolvedCountedText,
        )
        return Result.success(Unit)
    }

    suspend fun createCycleCount(
        locationId: String,
        productId: String,
        expectedQtyText: String? = null,
    ): Result<CycleCount> {
        val lid = locationId.trim()
        val pid = productId.trim()
        if (lid.isEmpty() || pid.isEmpty()) {
            return Result.failure(IllegalArgumentException("Location + product required"))
        }
        val expected: Long? = expectedQtyText?.trim()?.takeIf { it.isNotEmpty() }?.let { raw ->
            val parsed = raw.toLongOrNull()
            if (parsed == null || parsed < 0L) {
                return Result.failure(IllegalArgumentException("Expected qty must be >= 0"))
            }
            parsed
        }
        val res = opsRepository.createCycleCount(lid, pid, expected)
        val created = res.body()
        if (!res.isSuccessful || created == null) {
            return Result.failure(IllegalStateException("Create count failed (${res.code()})"))
        }
        state = state.copy(
            selectedCountId = created.countId,
            countedQtyText = created.expectedQty.toString(),
            cycleReasonCode = "",
        )
        refreshCycleCountsAndAdjustments()
        return Result.success(created)
    }

    suspend fun submitCycleCount(
        countId: String,
        countedQtyText: String,
        reasonCode: String? = null,
    ): Result<CycleCount> {
        val qty = TransferWmsHelpers.parseCountedQty(countedQtyText)
            .getOrElse { return Result.failure(it) }
        val res = opsRepository.submitCycleCount(countId, qty, reasonCode)
        val submitted = res.body()
        if (!res.isSuccessful || submitted == null) {
            return Result.failure(IllegalStateException("Submit failed (${res.code()})"))
        }
        state = state.copy(
            selectedCountId = null,
            countedQtyText = "",
            cycleReasonCode = "",
        )
        refreshCycleCountsAndAdjustments()
        return Result.success(submitted)
    }

    suspend fun approveAdjustment(adjustmentId: String): Result<InventoryAdjustment> {
        val res = opsRepository.approveInventoryAdjustment(adjustmentId)
        val updated = res.body()
        if (!res.isSuccessful || updated == null) {
            return Result.failure(IllegalStateException("Approve failed (${res.code()})"))
        }
        refreshCycleCountsAndAdjustments()
        return Result.success(updated)
    }

    suspend fun rejectAdjustment(adjustmentId: String): Result<InventoryAdjustment> {
        val res = opsRepository.rejectInventoryAdjustment(adjustmentId)
        val updated = res.body()
        if (!res.isSuccessful || updated == null) {
            return Result.failure(IllegalStateException("Reject failed (${res.code()})"))
        }
        refreshCycleCountsAndAdjustments()
        return Result.success(updated)
    }
}
