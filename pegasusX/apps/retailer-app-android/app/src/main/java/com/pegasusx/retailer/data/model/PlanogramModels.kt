package com.pegasusx.retailer.data.model

import com.google.gson.annotations.SerializedName

/**
 * Data models for Retail OS Pack 8 Planogram & Shelf Vision Architecture.
 */

data class PlanogramSlot(
    @SerializedName("slot_id") val slotId: String,
    @SerializedName("shelf_id") val shelfId: String,
    @SerializedName("col_index") val colIndex: Int,
    @SerializedName("expected_sku_id") val expectedSkuId: String,
    @SerializedName("sku_name") val skuName: String,
    @SerializedName("facings") val facings: Int = 1,
    @SerializedName("min_facings") val minFacings: Int = 1,
    @SerializedName("sku_source") val skuSource: String = "PEGASUS", // PEGASUS or LOCAL
    @SerializedName("width_mm") val widthMm: Int = 100,
    @SerializedName("height_mm") val heightMm: Int = 150,
)

data class PlanogramShelf(
    @SerializedName("shelf_id") val shelfId: String,
    @SerializedName("bay_id") val bayId: String,
    @SerializedName("row_index") val rowIndex: Int,
    @SerializedName("label") val label: String,
    @SerializedName("slots") val slots: List<PlanogramSlot> = emptyList(),
)

data class PlanogramBay(
    @SerializedName("bay_id") val bayId: String,
    @SerializedName("location_id") val locationId: String,
    @SerializedName("name") val name: String,
    @SerializedName("sort_order") val sortOrder: Int = 0,
    @SerializedName("shelves") val shelves: List<PlanogramShelf> = emptyList(),
)

data class PlanogramVersion(
    @SerializedName("version_id") val versionId: String,
    @SerializedName("location_id") val locationId: String,
    @SerializedName("status") val status: String, // DRAFT, PUBLISHED, ARCHIVED
    @SerializedName("published_at") val publishedAt: String? = null,
    @SerializedName("bays") val bays: List<PlanogramBay> = emptyList(),
)

data class ShelfAuditFinding(
    @SerializedName("finding_id") val findingId: String,
    @SerializedName("audit_id") val auditId: String,
    @SerializedName("slot_id") val slotId: String? = null,
    @SerializedName("type") val type: String, // GAP, WRONG_SKU, EMPTY, OK, UNKNOWN
    @SerializedName("expected_sku") val expectedSku: String,
    @SerializedName("detected_sku") val detectedSku: String? = null,
    @SerializedName("confidence") val confidence: Double = 0.0,
    @SerializedName("status") val status: String = "PENDING_REVIEW", // PENDING_REVIEW, ACCEPTED, DISMISSED
    @SerializedName("shelf_row_index") val shelfRowIndex: Int? = null,
    @SerializedName("slot_col_index") val slotColIndex: Int? = null,
)

data class ShelfAudit(
    @SerializedName("audit_id") val auditId: String,
    @SerializedName("location_id") val locationId: String,
    @SerializedName("bay_id") val bayId: String? = null,
    @SerializedName("mode") val mode: String = "VISION", // HUMAN, VISION
    @SerializedName("status") val status: String = "OPEN", // OPEN, IN_REVIEW, CLOSED
    @SerializedName("created_at") val createdAt: String,
    @SerializedName("findings") val findings: List<ShelfAuditFinding> = emptyList(),
)
