package com.pegasusx.supplier

import com.pegasusx.mobilekit.foundry.ActionExecutionRecord
import com.pegasusx.mobilekit.foundry.ActionExecutionRequest
import com.pegasusx.mobilekit.foundry.LotRecallRequest
import com.pegasusx.mobilekit.foundry.OntologyObjectDto
import com.pegasusx.mobilekit.foundry.RecallBlastRadiusReport
import com.pegasusx.mobilekit.foundry.ScenarioCommitResult
import com.pegasusx.mobilekit.foundry.ScenarioPatchRequest
import com.pegasusx.mobilekit.foundry.ScenarioWatermarkConflict
import com.pegasusx.mobilekit.foundry.SearchAroundRequest
import com.pegasusx.mobilekit.foundry.SearchAroundResponse
import com.pegasusx.supplier.ui.viewmodel.FoundrySandboxUiState
import com.pegasusx.supplier.ui.viewmodel.ScenarioPatchSummary
import kotlinx.serialization.json.JsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class FoundryScenarioTest {

    @Test
    fun testScenarioPatchRequestAndState() {
        val patch = ScenarioPatchRequest(
            cellId = "cell-uz",
            supplierId = "supplier-tashkent-1",
            objectType = "SKU",
            objectId = "sku-uz-cola-1l",
            operation = "SET",
            patchProperties = mapOf(
                "price_minor" to JsonPrimitive(1250000L),
                "safety_stock" to JsonPrimitive(450),
            ),
        )

        assertEquals("cell-uz", patch.cellId)
        assertEquals("SKU", patch.objectType)
        assertEquals("sku-uz-cola-1l", patch.objectId)
        assertEquals("SET", patch.operation)
        assertEquals(2, patch.patchProperties.size)

        val summary = ScenarioPatchSummary(
            objectType = patch.objectType,
            objectId = patch.objectId,
            key = "price_minor",
            value = "1250000",
        )
        val state = FoundrySandboxUiState(
            appliedPatches = listOf(summary),
        )
        assertEquals(1, state.appliedPatches.size)
        assertEquals("price_minor", state.appliedPatches.first().key)
    }

    @Test
    fun testScenarioOptimisticConcurrencyWatermarkConflict() {
        val conflict = ScenarioWatermarkConflict(
            cellId = "cell-uz",
            objectType = "INVENTORY_BATCH",
            objectId = "batch-uz-sugar-091",
            baseVersion = 4L,
            liveVersion = 7L,
        )

        // Live version advanced past branch watermark -> Conflict 409
        assertTrue(conflict.liveVersion > conflict.baseVersion)
        assertEquals(4L, conflict.baseVersion)
        assertEquals(7L, conflict.liveVersion)

        val commitResult = ScenarioCommitResult(
            scenarioId = "scenario-surge-plan-01",
            committedObjects = 0,
            conflicts = listOf(conflict),
        )
        assertEquals(1, commitResult.conflicts.size)
        assertEquals(0, commitResult.committedObjects)

        val state = FoundrySandboxUiState(
            commitResult = commitResult,
            hasConflict = commitResult.conflicts.isNotEmpty(),
            conflicts = commitResult.conflicts,
        )
        assertTrue(state.hasConflict)
        assertEquals(1, state.conflicts.size)
    }

    @Test
    fun testContaminatedLotRecallBlastRadiusLineage() {
        val req = LotRecallRequest(
            cellId = "cell-uz",
            supplierId = "supplier-tashkent-1",
            rawMaterialLotId = "LOT-SUGAR-2026-X8",
        )
        assertEquals("LOT-SUGAR-2026-X8", req.rawMaterialLotId)

        val blastRadius = RecallBlastRadiusReport(
            cellId = "cell-uz",
            supplierId = "supplier-tashkent-1",
            contaminatedRawMaterialLot = "LOT-SUGAR-2026-X8",
            affectedWorkOrders = listOf("WO-PROD-9021", "WO-PROD-9022"),
            affectedFinishedLots = listOf("FL-BEV-5510", "FL-BEV-5511"),
            quarantinedSsccPallets = listOf("003478901234567890", "003478901234567891"),
            quarantinedWarehouseTotes = listOf("tote-tash-01", "tote-tash-02"),
            inTransitDriverDropBlocks = listOf("drop-uz-4412"),
            impactedSoliqFacturas = listOf("FACTURA-2026-9901"),
        )

        assertEquals("LOT-SUGAR-2026-X8", blastRadius.contaminatedRawMaterialLot)
        assertEquals(2, blastRadius.affectedWorkOrders.size)
        assertEquals(2, blastRadius.affectedFinishedLots.size)
        assertEquals(2, blastRadius.quarantinedSsccPallets.size)
        assertEquals(1, blastRadius.inTransitDriverDropBlocks.size)
        assertEquals(1, blastRadius.impactedSoliqFacturas.size)
    }

    @Test
    fun testEmergencyQuarantineKineticAction() {
        val actionReq = ActionExecutionRequest(
            actionName = "EXECUTE_LOT_QUARANTINE_BLAST_RADIUS",
            actionType = "MUTATION",
            actorId = "supplier-tashkent-1",
            actorRole = "SUPPLIER_QUALITY_DIRECTOR",
            targetObjectType = "RAW_MATERIAL_LOT",
            targetObjectId = "LOT-SUGAR-2026-X8",
            payload = emptyMap(),
            idempotencyKey = "idem-recall-99012",
            cellId = "cell-uz",
            supplierId = "supplier-tashkent-1",
        )

        assertEquals("EXECUTE_LOT_QUARANTINE_BLAST_RADIUS", actionReq.actionName)
        assertEquals("MUTATION", actionReq.actionType)
        assertEquals("SUPPLIER_QUALITY_DIRECTOR", actionReq.actorRole)
        assertEquals("LOT-SUGAR-2026-X8", actionReq.targetObjectId)

        val record = ActionExecutionRecord(
            executionId = "exec-action-7721",
            actionName = actionReq.actionName,
            actionType = actionReq.actionType,
            actorId = actionReq.actorId,
            actorRole = actionReq.actorRole,
            targetObjectType = actionReq.targetObjectType,
            targetObjectId = actionReq.targetObjectId,
            idempotencyKey = actionReq.idempotencyKey,
            executedAt = "2026-10-30T14:35:00Z",
            status = "EXECUTED",
            cellId = "cell-uz",
            supplierId = "supplier-tashkent-1",
        )
        assertEquals("EXECUTED", record.status)
        assertEquals("exec-action-7721", record.executionId)
    }

    @Test
    fun testSearchAroundOntologyContract() {
        val searchReq = SearchAroundRequest(
            cellId = "cell-uz",
            startObjectType = "RAW_MATERIAL_LOT",
            startObjectIds = listOf("LOT-SUGAR-2026-X8"),
            allowedLinkTypes = listOf("CONSUMED_IN", "PRODUCED_LOT", "SHIPPED_IN_PALLET"),
            maxHops = 4,
        )
        assertEquals(4, searchReq.maxHops)
        assertEquals(3, searchReq.allowedLinkTypes.size)

        val obj = OntologyObjectDto(
            cellId = "cell-uz",
            objectType = "FINISHED_LOT",
            objectId = "FL-BEV-5510",
            supplierId = "supplier-tashkent-1",
            version = 2L,
        )
        val resp = SearchAroundResponse(objects = listOf(obj))
        assertEquals(1, resp.objects.size)
        assertEquals("FL-BEV-5510", resp.objects.first().objectId)
    }
}
