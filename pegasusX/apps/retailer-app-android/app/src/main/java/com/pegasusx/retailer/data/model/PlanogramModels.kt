package com.pegasusx.retailer.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Data models for Retail OS Pack 8 Planogram & Shelf Vision Architecture.
 */

@Serializable
data class PlanogramSlot(
    @SerialName("slot_id") val slotId: String,
    @SerialName("shelf_id") val shelfId: String,
    @SerialName("col_index") val colIndex: Int,
    @SerialName("expected_sku_id") val expectedSkuId: String,
    @SerialName("sku_name") val skuName: String,
    @SerialName("facings") val facings: Int = 1,
    @SerialName("min_facings") val minFacings: Int = 1,
    @SerialName("sku_source") val skuSource: String = "PEGASUS", // PEGASUS or LOCAL
    @SerialName("width_mm") val widthMm: Int = 100,
    @SerialName("height_mm") val heightMm: Int = 150,
)

@Serializable
data class PlanogramShelf(
    @SerialName("shelf_id") val shelfId: String,
    @SerialName("bay_id") val bayId: String,
    @SerialName("row_index") val rowIndex: Int,
    @SerialName("label") val label: String,
    @SerialName("slots") val slots: List<PlanogramSlot> = emptyList(),
)

@Serializable
data class PlanogramBay(
    @SerialName("bay_id") val bayId: String,
    @SerialName("location_id") val locationId: String,
    @SerialName("name") val name: String,
    @SerialName("sort_order") val sortOrder: Int = 0,
    @SerialName("shelves") val shelves: List<PlanogramShelf> = emptyList(),
)

@Serializable
data class PlanogramVersion(
    @SerialName("version_id") val versionId: String,
    @SerialName("location_id") val locationId: String,
    @SerialName("status") val status: String, // DRAFT, PUBLISHED, ARCHIVED
    @SerialName("published_at") val publishedAt: String? = null,
    @SerialName("bays") val bays: List<PlanogramBay> = emptyList(),
)

@Serializable
data class ShelfAuditFinding(
    @SerialName("finding_id") val findingId: String,
    @SerialName("audit_id") val auditId: String,
    @SerialName("slot_id") val slotId: String? = null,
    @SerialName("type") val type: String, // GAP, WRONG_SKU, EMPTY, OK, UNKNOWN
    @SerialName("expected_sku") val expectedSku: String,
    @SerialName("detected_sku") val detectedSku: String? = null,
    @SerialName("confidence") val confidence: Double = 0.0,
    @SerialName("status") val status: String = "PENDING_REVIEW", // PENDING_REVIEW, ACCEPTED, DISMISSED
    @SerialName("shelf_row_index") val shelfRowIndex: Int? = null,
    @SerialName("slot_col_index") val slotColIndex: Int? = null,
)

@Serializable
data class ShelfAudit(
    @SerialName("audit_id") val auditId: String,
    @SerialName("location_id") val locationId: String,
    @SerialName("bay_id") val bayId: String? = null,
    @SerialName("mode") val mode: String = "VISION", // HUMAN, VISION
    @SerialName("status") val status: String = "OPEN", // OPEN, IN_REVIEW, CLOSED
    @SerialName("created_at") val createdAt: String,
    @SerialName("findings") val findings: List<ShelfAuditFinding> = emptyList(),
)
