package com.pegasusx.supplier.data.model

import androidx.annotation.StringRes
import com.pegasusx.supplier.R
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import java.util.Locale

enum class MigrationPack(val rawValue: String, @StringRes val labelRes: Int) {
    WAREHOUSES("warehouses", R.string.portal_nav_warehouses),
    PRODUCTS_PRICES("products_prices", R.string.supplier_portal_migrations_pack_products),
    CUSTOMERS("customers", R.string.supplier_portal_migrations_pack_retailers),
    STOCK("stock", R.string.supplier_portal_migrations_pack_stock),
    LOTS("lots", R.string.supplier_portal_migrations_pack_lots),
    HISTORICAL_ORDERS("historical_orders", R.string.supplier_portal_migrations_pack_orders_history),
    NASIYA_AR("nasiya_ar", R.string.supplier_portal_migrations_pack_nasiya),
    OPEN_ORDERS("open_orders", R.string.supplier_portal_migrations_pack_orders_open),
    ;

    companion object {
        @StringRes
        fun labelResFor(raw: String?): Int =
            entries.firstOrNull { it.rawValue == raw }?.labelRes
                ?: R.string.supplier_portal_migrations_pack_quarantine
    }
}

object MigrationEncodingOptions {
    val all = listOf("utf-8", "windows-1251", "cp1251", "cp866")
}

@Serializable
data class MigrationProject(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("project_id") val projectId: String,
    val name: String = "",
    val mode: String = "FULL",
    val status: String = "",
    @SerialName("default_warehouse_id") val defaultWarehouseId: String? = null,
    @SerialName("created_by") val createdBy: String = "",
    @SerialName("cutover_at") val cutoverAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class MigrationCreateProjectRequest(
    val name: String,
    @SerialName("default_warehouse_id") val defaultWarehouseId: String? = null,
)

@Serializable
data class MigrationProjectsResponse(
    val projects: List<MigrationProject> = emptyList(),
)

@Serializable
data class MigrationSortedStreamDecision(
    @SerialName("artifact_id") val artifactId: String = "",
    @SerialName("file_name") val fileName: String = "",
    @SerialName("stream_key") val streamKey: String = "",
    val encoding: String = "utf-8",
    @SerialName("total_rows") val totalRows: Long = 0,
    val headers: List<String> = emptyList(),
    @SerialName("preview_raw") val previewRaw: List<String> = emptyList(),
    @SerialName("detected_packs") val detectedPacks: List<String> = emptyList(),
    @SerialName("split_keys") val splitKeys: List<String> = emptyList(),
    val confidence: Double = 0.0,
    @SerialName("is_composite") val isComposite: Boolean = false,
    val ignored: Boolean = false,
    @SerialName("needs_review") val needsReview: Boolean = false,
    @SerialName("review_reason") val reviewReason: String? = null,
) {
    val id: String get() = "$artifactId::$streamKey"
}

@Serializable
data class MigrationSortSummary(
    val streams: List<MigrationSortedStreamDecision> = emptyList(),
    @SerialName("conflicting_inventory_mode") val conflictingInventoryMode: Boolean = false,
    @SerialName("needs_review") val needsReview: Boolean = false,
    @SerialName("review_reasons") val reviewReasons: List<String> = emptyList(),
)

@Serializable
data class MigrationBundle(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("project_id") val projectId: String = "",
    @SerialName("bundle_id") val bundleId: String,
    val status: String = "",
    @SerialName("sort_summary") val sortSummary: MigrationSortSummary? = null,
    @SerialName("error_message") val errorMessage: String? = null,
    @SerialName("created_by") val createdBy: String = "",
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class MigrationArtifact(
    @SerialName("artifact_id") val artifactId: String,
    @SerialName("file_name") val fileName: String = "",
    @SerialName("byte_size") val byteSize: Long = 0,
    @SerialName("declared_mime") val declaredMime: String? = null,
    @SerialName("detected_mime") val detectedMime: String? = null,
    @SerialName("content_sha256") val contentSha256: String = "",
    val encoding: String? = null,
    @SerialName("gcs_path") val gcsPath: String = "",
)

@Serializable
data class MigrationBundleTree(
    val bundle: MigrationBundle,
    val artifacts: List<MigrationArtifact> = emptyList(),
)

@Serializable
data class MigrationSession(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("session_id") val sessionId: String,
    val status: String = "",
    @SerialName("file_name") val fileName: String = "",
    @SerialName("total_rows") val totalRows: Long = 0,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("bundle_id") val bundleId: String? = null,
    @SerialName("artifact_id") val artifactId: String? = null,
    @SerialName("stream_key") val streamKey: String? = null,
    @SerialName("stream_split_key") val streamSplitKey: String? = null,
    val pack: String? = null,
    @SerialName("sheet_name") val sheetName: String? = null,
    @SerialName("ai_model") val aiModel: String? = null,
    @SerialName("ai_status") val aiStatus: String? = null,
    @SerialName("dry_run_report_id") val dryRunReportId: String? = null,
    @SerialName("dry_run_fingerprint") val dryRunFingerprint: String? = null,
    @SerialName("apply_mode") val applyMode: String? = null,
    @SerialName("set_on_hand_confirmed") val setOnHandConfirmed: Boolean? = null,
    @SerialName("resume_row_index") val resumeRowIndex: Long? = null,
    @SerialName("rollback_until") val rollbackUntil: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class MigrationDAGStatus(
    val order: List<String> = emptyList(),
    @SerialName("runnable_now") val runnableNow: List<String> = emptyList(),
    @SerialName("blocked_by") val blockedBy: Map<String, List<String>> = emptyMap(),
    @SerialName("missing_packs") val missingPacks: List<String> = emptyList(),
)

@Serializable
data class MigrationProjectTree(
    val project: MigrationProject,
    val bundles: List<MigrationBundleTree> = emptyList(),
    val sessions: List<MigrationSession> = emptyList(),
    val dag: MigrationDAGStatus = MigrationDAGStatus(),
)

@Serializable
data class MigrationDeclareFile(
    @SerialName("file_name") val fileName: String,
    @SerialName("byte_size") val byteSize: Long,
    @SerialName("content_sha256") val contentSha256: String,
)

@Serializable
data class MigrationCreateBundleRequest(
    val files: List<MigrationDeclareFile>,
)

@Serializable
data class MigrationUploadTicket(
    @SerialName("artifact_id") val artifactId: String,
    @SerialName("file_name") val fileName: String = "",
    @SerialName("gcs_path") val gcsPath: String = "",
    @SerialName("upload_url") val uploadUrl: String,
    val method: String = "PUT",
    @SerialName("content_type") val contentType: String = "application/octet-stream",
    val resumable: Boolean = false,
    @SerialName("required_headers") val requiredHeaders: Map<String, String> = emptyMap(),
    @SerialName("expires_at") val expiresAt: String = "",
)

@Serializable
data class MigrationCreateBundleResponse(
    val bundle: MigrationBundle,
    val artifacts: List<MigrationArtifact> = emptyList(),
    @SerialName("upload_tickets") val uploadTickets: List<MigrationUploadTicket> = emptyList(),
)

@Serializable
data class MigrationUploadedArtifactMeta(
    @SerialName("byte_size") val byteSize: Long,
    @SerialName("content_sha256") val contentSha256: String,
)

@Serializable
data class MigrationBundleUploadedRequest(
    val artifacts: Map<String, MigrationUploadedArtifactMeta>,
)

@Serializable
data class MigrationBundleUploadedResponse(
    val bundle: MigrationBundle,
    val summary: MigrationSortSummary? = null,
    val sessions: List<MigrationSession> = emptyList(),
)

@Serializable
data class MigrationStreamOverrideItem(
    @SerialName("artifact_id") val artifactId: String,
    @SerialName("stream_key") val streamKey: String,
    val packs: List<String>? = null,
    val ignore: Boolean? = null,
    val encoding: String? = null,
    val confirmed: Boolean? = null,
)

@Serializable
data class MigrationBundleSortOverrideRequest(
    val overrides: List<MigrationStreamOverrideItem>,
    @SerialName("resolved_inventory_mode") val resolvedInventoryMode: String? = null,
    @SerialName("confirm_review") val confirmReview: Boolean = true,
)

@Serializable
data class MigrationBundleSortOverrideResponse(
    @SerialName("bundle_id") val bundleId: String,
    @SerialName("bundle_status") val bundleStatus: String = "",
    val summary: MigrationSortSummary = MigrationSortSummary(),
    val sessions: List<MigrationSession> = emptyList(),
)

@Serializable
data class MigrationCanonicalField(
    val key: String,
    @SerialName("label_en") val labelEn: String = "",
    @SerialName("label_ru") val labelRu: String? = null,
    @SerialName("label_uz") val labelUz: String? = null,
    val required: Boolean = false,
    val type: String = "string",
) {
    fun localizedLabel(language: String = Locale.getDefault().language): String =
        when (language) {
            "ru" -> labelRu?.takeIf { it.isNotBlank() } ?: labelEn
            "uz" -> labelUz?.takeIf { it.isNotBlank() } ?: labelEn
            else -> labelEn
        }.ifBlank { key }
}

@Serializable
data class MigrationColumnMappingProposal(
    @SerialName("source_column") val sourceColumn: String,
    @SerialName("target_field") val targetField: String = "",
    val confidence: Double = 0.0,
    val reason: String? = null,
    val source: String? = null,
    val confirmed: Boolean = false,
    val transform: String? = null,
)

@Serializable
data class MigrationSessionMappingDocument(
    @SerialName("session_id") val sessionId: String,
    val pack: String = "",
    @SerialName("header_fingerprint") val headerFingerprint: String = "",
    @SerialName("ai_model") val aiModel: String = "",
    @SerialName("ai_status") val aiStatus: String = "",
    val mappings: List<MigrationColumnMappingProposal> = emptyList(),
    @SerialName("missing_required") val missingRequired: List<String> = emptyList(),
    @SerialName("low_confidence_columns") val lowConfidenceColumns: List<String> = emptyList(),
    @SerialName("all_required_mapped") val allRequiredMapped: Boolean = false,
    @SerialName("mapping_hash") val mappingHash: String = "",
)

@Serializable
data class MigrationGetSessionMappingResponse(
    val session: MigrationSession,
    val taxonomy: List<MigrationCanonicalField> = emptyList(),
    val mapping: MigrationSessionMappingDocument,
)

@Serializable
data class MigrationConfirmMappingRequest(
    val mappings: List<MigrationColumnMappingProposal>? = null,
    @SerialName("accept_above_confidence") val acceptAboveConfidence: Double? = null,
    @SerialName("save_to_library") val saveToLibrary: Boolean? = null,
    val strict: Boolean = true,
)

@Serializable
data class MigrationConfirmMappingResponse(
    val session: MigrationSession,
    val mapping: MigrationSessionMappingDocument,
)

@Serializable
data class MigrationStagedRow(
    @SerialName("row_index") val rowIndex: Long,
    @SerialName("raw_data") val rawData: Map<String, JsonElement> = emptyMap(),
    @SerialName("validation_errors") val validationErrors: List<String> = emptyList(),
) {
    fun textValue(column: String): String {
        val el = rawData[column] ?: return ""
        return when (el) {
            is JsonNull -> ""
            is JsonPrimitive -> el.content
            else -> el.toString()
        }
    }
}

@Serializable
data class MigrationSessionRowsResponse(
    @SerialName("session_id") val sessionId: String = "",
    @SerialName("total_rows") val totalRows: Long = 0,
    val rows: List<MigrationStagedRow> = emptyList(),
)

@Serializable
data class MigrationDryRunConflict(
    @SerialName("session_id") val sessionId: String = "",
    val pack: String = "",
    @SerialName("row_index") val rowIndex: Long = 0,
    val code: String = "",
    val severity: String = "BLOCKING",
    val field: String? = null,
    val message: String = "",
    val suggestion: String? = null,
)

@Serializable
data class MigrationPackDryRunReport(
    @SerialName("session_id") val sessionId: String,
    val pack: String = "",
    @SerialName("rows_total") val rowsTotal: Long = 0,
    @SerialName("rows_valid") val rowsValid: Long = 0,
    @SerialName("rows_invalid") val rowsInvalid: Long = 0,
    @SerialName("would_create") val wouldCreate: Long = 0,
    @SerialName("would_update") val wouldUpdate: Long = 0,
    @SerialName("would_skip") val wouldSkip: Long = 0,
    @SerialName("money_impact_minor") val moneyImpactMinor: Long = 0,
    val currency: String = "UZS",
    @SerialName("blocking_count") val blockingCount: Long = 0,
    @SerialName("warning_count") val warningCount: Long = 0,
    @SerialName("cold_lot_migrated_without_live_baseline") val coldLotWithoutLive: Long? = null,
)

@Serializable
data class MigrationDryRunSummary(
    @SerialName("project_id") val projectId: String = "",
    @SerialName("content_fingerprint") val contentFingerprint: String = "",
    val packs: List<MigrationPackDryRunReport> = emptyList(),
    val conflicts: List<MigrationDryRunConflict> = emptyList(),
    @SerialName("balanced_journal_check") val balancedJournalCheck: Boolean = false,
    @SerialName("journal_debit_minor") val journalDebitMinor: Long = 0,
    @SerialName("journal_credit_minor") val journalCreditMinor: Long = 0,
    @SerialName("opening_balance_account") val openingBalanceAccount: String = "",
    @SerialName("ready_to_approve") val readyToApprove: Boolean = false,
    @SerialName("cold_lot_migrated_without_live_baseline") val coldLotWarningCount: Long = 0,
    @SerialName("on_hand_below_reserved") val onHandBelowReservedHits: Long = 0,
    @SerialName("evaluated_at") val evaluatedAt: String = "",
    @SerialName("expires_at") val expiresAt: String = "",
)

@Serializable
data class MigrationReport(
    @SerialName("report_id") val reportId: String,
    @SerialName("project_id") val projectId: String? = null,
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("report_kind") val reportKind: String = "",
    @SerialName("rows_total") val rowsTotal: Long = 0,
    @SerialName("rows_valid") val rowsValid: Long = 0,
    @SerialName("rows_invalid") val rowsInvalid: Long = 0,
    @SerialName("would_create") val wouldCreate: Long = 0,
    @SerialName("would_update") val wouldUpdate: Long = 0,
    @SerialName("would_skip") val wouldSkip: Long = 0,
    @SerialName("money_impact_minor") val moneyImpactMinor: Long = 0,
    val currency: String = "UZS",
    @SerialName("blocking_count") val blockingCount: Long = 0,
    @SerialName("warning_count") val warningCount: Long = 0,
)

@Serializable
data class MigrationDryRunRequest(
    @SerialName("apply_mode") val applyMode: String,
)

@Serializable
data class MigrationDryRunResponse(
    val report: MigrationReport,
    val summary: MigrationDryRunSummary,
)

@Serializable
data class MigrationApproveRequest(
    @SerialName("apply_mode") val applyMode: String? = null,
    @SerialName("set_on_hand_confirmed") val setOnHandConfirmed: Boolean? = null,
    @SerialName("expected_report_id") val expectedReportId: String? = null,
)

@Serializable
data class MigrationApproveResponse(
    val session: MigrationSession,
)

@Serializable
data class MigrationApplyRequest(
    @SerialName("apply_mode") val applyMode: String,
    @SerialName("chunk_size") val chunkSize: Int = 500,
)

@Serializable
data class MigrationApplyWarning(
    @SerialName("row_index") val rowIndex: Long = 0,
    val code: String = "",
    val message: String = "",
)

@Serializable
data class MigrationApplyResult(
    val session: MigrationSession,
    @SerialName("already_applied") val alreadyApplied: Boolean = false,
    val created: Long = 0,
    val updated: Long = 0,
    val skipped: Long = 0,
    @SerialName("chunks_applied") val chunksApplied: Long = 0,
    @SerialName("last_row_index") val lastRowIndex: Long = 0,
    val warnings: List<MigrationApplyWarning> = emptyList(),
)

@Serializable
data class MigrationRollbackResult(
    val session: MigrationSession,
    @SerialName("reverted_snapshots") val revertedSnapshots: Long = 0,
    @SerialName("reversed_invoices") val reversedInvoices: Long = 0,
    @SerialName("superseded_orders") val supersededOrders: Long = 0,
    @SerialName("reversed_amount_minor") val reversedAmountMinor: Long = 0,
)

@Serializable
data class MigrationOpenOrderReleaseRequest(
    @SerialName("order_ids") val orderIds: List<String>,
    @SerialName("target_status") val targetStatus: String,
)

@Serializable
data class MigrationOpenOrderReleaseResponse(
    @SerialName("released_order_ids") val releasedOrderIds: List<String> = emptyList(),
    @SerialName("skipped_order_ids") val skippedOrderIds: List<String> = emptyList(),
    val errors: Map<String, String> = emptyMap(),
)

@Serializable
data class MigrationCutoverChecklistItem(
    val name: String = "",
    val passed: Boolean = false,
    val detail: String = "",
)

@Serializable
data class MigrationCutoverResult(
    val project: MigrationProject,
    val sessions: List<MigrationSession> = emptyList(),
    val checklist: List<MigrationCutoverChecklistItem> = emptyList(),
    @SerialName("cutover_at") val cutoverAt: String = "",
)

@Serializable
data class INNLookupRequest(
    val inn: String,
    @SerialName("tenant_type") val tenantType: String = "SUPPLIER",
)

@Serializable
data class INNLookupResponse(
    val status: String,
    @SerialName("normalized_inn") val normalizedInn: String = "",
    @SerialName("entity_type") val entityType: String? = null,
    @SerialName("legal_name") val legalName: String? = null,
    val address: String? = null,
    @SerialName("registry_state") val registryState: String? = null,
    val cached: Boolean = false,
    @SerialName("existing_tenant_id") val existingTenantId: String? = null,
    @SerialName("existing_tenant_type") val existingTenantType: String? = null,
    @SerialName("existing_legal_name") val existingLegalName: String? = null,
    @SerialName("existing_join_eligible") val existingJoinEligible: Boolean = false,
)

@Serializable
data class JoinRequestConflictPayload(
    val error: String = "",
    @SerialName("request_id") val requestId: String? = null,
    @SerialName("target_tenant_id") val targetTenantId: String? = null,
    @SerialName("legal_name") val legalName: String? = null,
    @SerialName("normalized_inn") val normalizedInn: String? = null,
)

@Serializable
data class JoinRequestItem(
    @SerialName("request_id") val requestId: String,
    @SerialName("home_cell") val homeCell: String = "",
    @SerialName("target_tenant_type") val targetTenantType: String = "",
    @SerialName("target_tenant_id") val targetTenantId: String = "",
    @SerialName("normalized_inn") val normalizedInn: String = "",
    @SerialName("requester_phone") val requesterPhone: String = "",
    @SerialName("requester_name") val requesterName: String? = null,
    @SerialName("requester_email") val requesterEmail: String? = null,
    @SerialName("role_sought") val roleSought: String = "",
    val status: String = "",
    @SerialName("decided_by") val decidedBy: String? = null,
    @SerialName("decided_at") val decidedAt: String? = null,
    @SerialName("decision_reason") val decisionReason: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class JoinRequestsResponse(
    @SerialName("supplier_id") val supplierId: String? = null,
    val requests: List<JoinRequestItem> = emptyList(),
)

@Serializable
data class SupplierARInvoice(
    @SerialName("invoice_id") val invoiceId: String,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("order_id") val orderId: String = "",
    val status: String = "",
    @SerialName("principal_minor") val principalMinor: Long = 0,
    @SerialName("balance_minor") val balanceMinor: Long = 0,
    val currency: String = "UZS",
    @SerialName("due_at") val dueAt: String? = null,
    val source: String? = null,
    @SerialName("external_id") val externalId: String? = null,
    @SerialName("acknowledged_at") val acknowledgedAt: String? = null,
    @SerialName("disputed_at") val disputedAt: String? = null,
    @SerialName("dispute_reason") val disputeReason: String? = null,
) {
    val isMigrationOrigin: Boolean
        get() = source.equals("MIGRATION", ignoreCase = true)

    val ackState: String
        get() = when {
            !disputedAt.isNullOrBlank() -> "DISPUTED"
            !acknowledgedAt.isNullOrBlank() -> "ACKNOWLEDGED"
            else -> "PENDING_ACK"
        }
}

@Serializable
data class SupplierARInvoicesResponse(
    val invoices: List<SupplierARInvoice> = emptyList(),
)

data class MigrationApplyProgressEvent(
    val sessionId: String,
    val processedRows: Long,
    val totalRows: Long,
    val chunkId: String? = null,
)
