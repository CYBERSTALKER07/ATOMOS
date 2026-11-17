package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.model.ProblemDetail
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.Variant
import java.io.IOException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import retrofit2.HttpException

enum class PackSnapMode {
    NEAREST,
    FLOOR,
    CEIL,
}

data class PackMultipleSuggestion(
    val skuId: String,
    val productName: String? = null,
    val packSize: Int,
    val requestedQty: Int? = null,
    val suggestedFloor: Int,
    val suggestedCeil: Int,
    val suggestedNearest: Int,
    val message: String = "",
)

enum class PendingOrderConflictKind {
    OWNER_APPROVAL_REQUIRED,
    REQUEST_IN_PROGRESS,
    RATE_LIMITED,
    IDEMPOTENT_ALREADY_CREATED,
    REJECTED_VALIDATION,
    UNAUTHORIZED,
    TRANSIENT_RETRY,
    PERMANENT_CLIENT_ERROR,
}

data class PendingOrderErrorAssessment(
    val kind: PendingOrderConflictKind,
    val status: String,
    val message: String,
    val shouldDelete: Boolean,
    val shouldNotify: Boolean,
    val retryable: Boolean,
)

object OrderLineContractHelper {
    val lenientJson: Json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
        encodeDefaults = true
    }

    /**
     * Snaps [quantity] to a valid multiple of [packSize], matching backend
     * `order/pack_multiples.go:SnapToPackMultiple`.
     */
    fun snapToPackMultiple(
        quantity: Int,
        packSize: Int,
        mode: PackSnapMode = PackSnapMode.NEAREST,
    ): Int {
        if (quantity <= 0) return 0
        if (packSize <= 1) return quantity
        val remainder = quantity % packSize
        if (remainder == 0) return quantity
        val lower = quantity - remainder
        val upper = lower + packSize
        return when (mode) {
            PackSnapMode.FLOOR -> lower
            PackSnapMode.CEIL -> upper
            PackSnapMode.NEAREST -> when {
                remainder * 2 >= packSize -> upper
                lower == 0 -> upper
                else -> lower
            }
        }
    }

    fun effectivePackSize(product: Product, variant: Variant): Int {
        if (variant.packCount > 1) return variant.packCount
        val blockUnits = product.unitsPerBlock ?: 1
        if (product.sellByBlock && blockUnits > 1) return blockUnits
        return 1
    }

    /**
     * Formats signed price delta from minor units (tiyin) into rounded integer `so'm`
     * without IEEE-754 float intermediates or `-0` artifacts.
     */
    fun formatPriceGapSom(gapMinor: Long, currencyLabel: String = "so'm"): String {
        val label = currencyLabel.trim().ifEmpty { "so'm" }.let {
            if (it.equals("UZS", ignoreCase = true)) "so'm" else it
        }
        val negative = gapMinor < 0L
        val absMinor = if (gapMinor == Long.MIN_VALUE) Long.MAX_VALUE else kotlin.math.abs(gapMinor)
        val roundedUnits = (absMinor + 50L) / 100L
        if (roundedUnits == 0L) return "0 $label"
        val grouped = groupThousands(roundedUnits.toString())
        val sign = if (negative) "-" else "+"
        return "$sign$grouped $label"
    }

    fun formatWholeSom(minor: Long, currencyLabel: String = "so'm"): String {
        val label = currencyLabel.trim().ifEmpty { "so'm" }.let {
            if (it.equals("UZS", ignoreCase = true)) "so'm" else it
        }
        val negative = minor < 0L
        val absMinor = if (minor == Long.MIN_VALUE) Long.MAX_VALUE else kotlin.math.abs(minor)
        val roundedUnits = (absMinor + 50L) / 100L
        if (roundedUnits == 0L) return "0 $label"
        val grouped = groupThousands(roundedUnits.toString())
        val sign = if (negative) "-" else ""
        return "$sign$grouped $label"
    }

    fun parsePackMultipleSuggestions(
        error: Throwable,
        fallbackSku: String? = null,
    ): List<PackMultipleSuggestion> {
        return when (error) {
            is ProblemDetailException -> parsePackMultipleSuggestions(error.problem, fallbackSku)
            is HttpException -> {
                val rawBody = RetailerW2ContractHelper.peekHttpErrorBody(error)
                parsePackMultipleSuggestions(rawBody, fallbackSku)
            }
            else -> parsePackMultipleSuggestions(error.message, fallbackSku)
        }
    }

    fun parsePackMultipleSuggestions(
        problem: ProblemDetail,
        fallbackSku: String? = null,
    ): List<PackMultipleSuggestion> {
        val fromRaw = parsePackMultipleSuggestions(problem.rawBody, fallbackSku)
        if (fromRaw.isNotEmpty()) return fromRaw

        val code = problem.code ?: problem.error
        if (!code.equals("PACK_MULTIPLE_REQUIRED", ignoreCase = true)) return emptyList()
        val packSize = (problem.packSize ?: 1).coerceAtLeast(1)
        val requested = problem.requested
        val floor = problem.suggestedFloor ?: requested?.let { snapToPackMultiple(it, packSize, PackSnapMode.FLOOR) } ?: 0
        val ceil = problem.suggestedCeil ?: requested?.let { snapToPackMultiple(it, packSize, PackSnapMode.CEIL) } ?: packSize
        val nearest = problem.suggestedNearest ?: requested?.let { snapToPackMultiple(it, packSize, PackSnapMode.NEAREST) } ?: ceil
        return listOf(
            PackMultipleSuggestion(
                skuId = problem.skuId ?: fallbackSku.orEmpty(),
                packSize = packSize,
                requestedQty = requested,
                suggestedFloor = floor,
                suggestedCeil = ceil,
                suggestedNearest = nearest,
                message = problem.detail ?: problem.title ?: "Quantity must be a multiple of $packSize",
            ),
        )
    }

    fun parsePackMultipleSuggestions(
        rawBody: String?,
        fallbackSku: String? = null,
    ): List<PackMultipleSuggestion> {
        val trimmed = rawBody?.trim().orEmpty()
        if (trimmed.isEmpty() || !trimmed.startsWith("{")) return emptyList()
        val root = runCatching { lenientJson.parseToJsonElement(trimmed) as? JsonObject }.getOrNull()
            ?: return emptyList()
        val topCode = root.str("code") ?: root.str("error") ?: ""
        val topMessage = root.str("message") ?: root.str("detail") ?: ""

        val listCandidate: JsonArray? = listOf("line_errors", "details", "items", "suggestions", "pack_errors", "lines")
            .firstNotNullOfOrNull { key -> root[key] as? JsonArray }

        if (listCandidate != null && listCandidate.isNotEmpty()) {
            val parsed = listCandidate.mapNotNull { entry ->
                (entry as? JsonObject)?.toPackSuggestion(topCode, topMessage, fallbackSku)
            }
            if (parsed.isNotEmpty()) return parsed
        }

        if (!topCode.equals("PACK_MULTIPLE_REQUIRED", ignoreCase = true) &&
            root["pack_size"] == null &&
            root["suggested_nearest"] == null &&
            root["suggested_ceil"] == null
        ) {
            return emptyList()
        }

        return listOfNotNull(root.toPackSuggestion(topCode, topMessage, fallbackSku))
    }

    fun assessPendingOrderError(error: Throwable): PendingOrderErrorAssessment {
        return when (error) {
            is ProblemDetailException -> {
                val p = error.problem
                val rawCodes = listOfNotNull(p.code, p.error, extractCodeFromRaw(p.rawBody)).joinToString(" ")
                val detail = p.detail ?: p.title ?: p.code ?: "Error ${p.status}"
                classifyHttpStatusAndCode(p.status, rawCodes, p.rawBody ?: detail)
            }
            is HttpException -> {
                val code = error.code()
                val rawBody = RetailerW2ContractHelper.peekHttpErrorBody(error).orEmpty()
                val bodyCode = extractCodeFromRaw(rawBody)
                val fallbackMsg = error.message().ifBlank { "HTTP $code" }
                classifyHttpStatusAndCode(code, bodyCode, rawBody.ifBlank { fallbackMsg })
            }
            is IOException -> PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.TRANSIENT_RETRY,
                status = "RETRY_QUEUED",
                message = error.message ?: "Offline network error — will retry when connected",
                shouldDelete = false,
                shouldNotify = false,
                retryable = true,
            )
            else -> PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.TRANSIENT_RETRY,
                status = "RETRY_QUEUED",
                message = error.message ?: "Sync failed — retry scheduled",
                shouldDelete = false,
                shouldNotify = false,
                retryable = true,
            )
        }
    }

    fun classifyHttpStatusAndCode(
        httpStatus: Int,
        codeHint: String,
        bodyOrMessage: String,
    ): PendingOrderErrorAssessment {
        val combined = "$codeHint $bodyOrMessage".lowercase()

        if (httpStatus == 429 || combined.contains("rate_limit_exceeded")) {
            return PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.RATE_LIMITED,
                status = "RATE_LIMITED",
                message = "Rate limited (429) — order kept in queue and will retry automatically.",
                shouldDelete = false,
                shouldNotify = true,
                retryable = true,
            )
        }

        if (httpStatus == 409) {
            if (combined.contains("owner_approval_required") || combined.contains("pending_owner")) {
                return PendingOrderErrorAssessment(
                    kind = PendingOrderConflictKind.OWNER_APPROVAL_REQUIRED,
                    status = "AWAITING_OWNER_APPROVAL",
                    message = "Owner approval required — order parked in sync queue.",
                    shouldDelete = false,
                    shouldNotify = true,
                    retryable = true,
                )
            }
            if (combined.contains("request_in_progress")) {
                return PendingOrderErrorAssessment(
                    kind = PendingOrderConflictKind.REQUEST_IN_PROGRESS,
                    status = "IN_PROGRESS",
                    message = "Order request is in progress on server — retrying shortly.",
                    shouldDelete = false,
                    shouldNotify = true,
                    retryable = true,
                )
            }
            if (combined.contains("idempotency_key_payload_mismatch") ||
                combined.contains("all_items_out_of_stock") ||
                combined.contains("partial_out_of_stock_rejected") ||
                combined.contains("pack_multiple_required") ||
                combined.contains("inventory_exhausted")
            ) {
                return PendingOrderErrorAssessment(
                    kind = PendingOrderConflictKind.REJECTED_VALIDATION,
                    status = "REJECTED",
                    message = "Queued order requires cart adjustment ($codeHint). Removed from auto-retry.",
                    shouldDelete = true,
                    shouldNotify = true,
                    retryable = false,
                )
            }
            return PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.IDEMPOTENT_ALREADY_CREATED,
                status = "SYNCED_IDEMPOTENT",
                message = "Order already recorded on server (409 idempotent).",
                shouldDelete = true,
                shouldNotify = true,
                retryable = false,
            )
        }

        if (httpStatus == 401) {
            return PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.UNAUTHORIZED,
                status = "AUTH_REQUIRED",
                message = "Session expired (401) — sign in again to submit your preserved order.",
                shouldDelete = false,
                shouldNotify = true,
                retryable = false,
            )
        }

        if (httpStatus == 408 || httpStatus in 500..599) {
            return PendingOrderErrorAssessment(
                kind = PendingOrderConflictKind.TRANSIENT_RETRY,
                status = "RETRY_QUEUED",
                message = "Server error ($httpStatus) — retry scheduled.",
                shouldDelete = false,
                shouldNotify = false,
                retryable = true,
            )
        }

        return PendingOrderErrorAssessment(
            kind = PendingOrderConflictKind.PERMANENT_CLIENT_ERROR,
            status = "DISCARDED",
            message = "Client error ($httpStatus): ${bodyOrMessage.take(120)}",
            shouldDelete = true,
            shouldNotify = true,
            retryable = false,
        )
    }

    private fun extractCodeFromRaw(rawBody: String?): String {
        val trimmed = rawBody?.trim().orEmpty()
        if (!trimmed.startsWith("{")) return trimmed
        val root = runCatching { lenientJson.parseToJsonElement(trimmed) as? JsonObject }.getOrNull()
            ?: return trimmed
        return listOfNotNull(
            root.str("code"),
            root.str("error"),
            root.str("status"),
            root.str("reason"),
            root.str("message"),
        ).joinToString(" ")
    }

    private fun JsonObject.toPackSuggestion(
        topCode: String,
        topMessage: String,
        fallbackSku: String?,
    ): PackMultipleSuggestion? {
        val entryCode = str("code") ?: str("error") ?: topCode
        val packSize = intField("pack_size") ?: intField("packSize") ?: intField("units_per_block")
        val hasSuggested = intField("suggested_nearest") != null ||
            intField("suggested_floor") != null ||
            intField("suggested_ceil") != null
        if (!entryCode.equals("PACK_MULTIPLE_REQUIRED", ignoreCase = true) && packSize == null && !hasSuggested) {
            return null
        }
        val effectivePack = (packSize ?: 1).coerceAtLeast(1)
        val requested = intField("requested") ?: intField("requested_qty") ?: intField("quantity")
        val floor = intField("suggested_floor")
            ?: requested?.let { snapToPackMultiple(it, effectivePack, PackSnapMode.FLOOR) }
            ?: 0
        val ceil = intField("suggested_ceil")
            ?: requested?.let { snapToPackMultiple(it, effectivePack, PackSnapMode.CEIL) }
            ?: effectivePack
        val nearest = intField("suggested_nearest")
            ?: requested?.let { snapToPackMultiple(it, effectivePack, PackSnapMode.NEAREST) }
            ?: ceil
        val skuId = str("sku_id") ?: str("sku") ?: str("product_id") ?: fallbackSku.orEmpty()
        val productName = str("product_name") ?: str("sku_name") ?: str("name")
        val message = str("message") ?: str("detail") ?: topMessage.ifBlank {
            "Quantity must be a multiple of $effectivePack"
        }
        return PackMultipleSuggestion(
            skuId = skuId,
            productName = productName,
            packSize = effectivePack,
            requestedQty = requested,
            suggestedFloor = floor,
            suggestedCeil = ceil,
            suggestedNearest = nearest,
            message = message,
        )
    }

    private fun JsonObject.str(key: String): String? = (this[key] as? JsonPrimitive)?.contentOrNull

    private fun JsonObject.intField(key: String): Int? {
        val p = this[key] as? JsonPrimitive ?: return null
        return p.intOrNull ?: p.contentOrNull?.toIntOrNull()
    }

    private fun groupThousands(digits: String): String {
        if (digits.length <= 3) return digits
        return buildString(digits.length + digits.length / 3) {
            digits.forEachIndexed { idx, ch ->
                if (idx > 0 && (digits.length - idx) % 3 == 0) append(' ')
                append(ch)
            }
        }
    }
}
