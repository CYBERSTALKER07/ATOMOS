package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.model.ProblemDetailException
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import retrofit2.HttpException

data class OwnerApprovalRequiredDetail(
    val requestId: String = "",
    val clientOrderId: String? = null,
    val status: String = "PENDING_OWNER",
    val trippedRule: String = "SELLER_MANDATE",
    val amountMinor: Long = 0L,
    val thresholdMinor: Long? = null,
    val remainingCapMinor: Long? = null,
    val message: String = "Order exceeds seller mandate and requires owner approval.",
    val fallbackPaymentMethods: List<String> = emptyList(),
) {
    val allowsCashFallback: Boolean
        get() = fallbackPaymentMethods.any { it.equals("CASH", ignoreCase = true) }

    val allowsCardFallback: Boolean
        get() = fallbackPaymentMethods.any { it.equals("CARD", ignoreCase = true) }

    val formattedAmount: String
        get() = OrderLineContractHelper.formatWholeSom(amountMinor)

    val formattedThreshold: String?
        get() = thresholdMinor?.let { OrderLineContractHelper.formatWholeSom(it) }

    val formattedRemainingCap: String?
        get() = remainingCapMinor?.let { OrderLineContractHelper.formatWholeSom(it) }

    val readableRuleLabel: String
        get() = RetailerW2ContractHelper.formatTrippedRuleLabel(trippedRule)
}

object RetailerW2ContractHelper {

    fun peekHttpErrorBody(error: HttpException): String? = runCatching {
        val body = error.response()?.errorBody() ?: return@runCatching null
        val peeked = body.source().peek()
        peeked.request(1_048_576L)
        peeked.readUtf8()
    }.getOrNull()

    fun formatTrippedRuleLabel(rule: String): String = when (rule.trim().uppercase()) {
        "DAILY_CREDIT_CAP_EXCEEDED" -> "Daily credit cap exceeded"
        "CREDIT_MANDATE_DISABLED" -> "Credit purchases disabled by seller mandate"
        "CASH_MANDATE_DISABLED" -> "Cash purchases disabled by seller mandate"
        "DAILY_CASH_CAP_EXCEEDED" -> "Daily cash cap exceeded"
        "MAX_SINGLE_ORDER_EXCEEDED" -> "Single-order mandate limit exceeded"
        else -> rule.trim().ifEmpty { "SELLER_MANDATE" }.replace('_', ' ').lowercase()
            .replaceFirstChar { it.uppercase() }
    }

    fun somInputToMinor(somText: String): Long? {
        val cleaned = somText.replace(Regex("[\\s,_'\\u00A0]"), "").trim()
        if (cleaned.isEmpty() || !cleaned.all { it.isDigit() }) return null
        val som = cleaned.toLongOrNull() ?: return null
        if (som < 0L || som > Long.MAX_VALUE / 100L) return null
        return som * 100L
    }

    fun minorToSomInput(minor: Long): String {
        if (minor <= 0L) return "0"
        return ((minor + 50L) / 100L).toString()
    }

    fun parseOwnerApprovalRequired(error: Throwable): OwnerApprovalRequiredDetail? {
        val status: Int?
        var jsonObj: JsonObject? = null
        var fallbackMessage: String? = null
        var fallbackCode: String? = null

        when (error) {
            is ProblemDetailException -> {
                status = error.problem.status
                fallbackMessage = error.problem.detail ?: error.problem.title
                fallbackCode = error.problem.code ?: error.problem.error
                jsonObj = error.problem.rawBody?.let { parseObject(it) }
            }
            is HttpException -> {
                status = error.code()
                val raw = peekHttpErrorBody(error)
                jsonObj = raw?.let { parseObject(it) }
                fallbackMessage = raw ?: error.message()
            }
            else -> return null
        }

        if (status != 409) return null
        val code = (jsonObj?.stringValue("code")
            ?: jsonObj?.stringValue("error")
            ?: fallbackCode
            ?: "")
            .trim()
            .lowercase()
        if (code != "owner_approval_required") return null

        val methods = jsonObj?.get("fallback_payment_methods")
            ?.let { it as? JsonArray }
            ?.mapNotNull { it.jsonPrimitive.contentOrNull?.trim()?.uppercase() }
            ?.filter { it.isNotEmpty() }
            ?: emptyList()

        return OwnerApprovalRequiredDetail(
            requestId = jsonObj?.stringValue("request_id").orEmpty(),
            clientOrderId = jsonObj?.stringValue("client_order_id"),
            status = jsonObj?.stringValue("status")?.ifBlank { null } ?: "PENDING_OWNER",
            trippedRule = jsonObj?.stringValue("tripped_rule")?.ifBlank { null } ?: "SELLER_MANDATE",
            amountMinor = jsonObj?.longValue("amount_minor") ?: 0L,
            thresholdMinor = jsonObj?.longValue("threshold_minor"),
            remainingCapMinor = jsonObj?.longValue("remaining_cap_minor"),
            message = jsonObj?.stringValue("message")?.ifBlank { null }
                ?: jsonObj?.stringValue("detail")?.ifBlank { null }
                ?: fallbackMessage?.takeIf { it.isNotBlank() }
                ?: "Order exceeds seller mandate and requires owner approval.",
            fallbackPaymentMethods = methods,
        )
    }

    fun resolveClaimErrorCode(error: Throwable): String? {
        var jsonObj: JsonObject? = null
        val rawText: String = when (error) {
            is ProblemDetailException -> {
                jsonObj = error.problem.rawBody?.let { parseObject(it) }
                val explicit = error.problem.code ?: error.problem.error
                if (!explicit.isNullOrBlank()) return explicit.trim().lowercase()
                error.problem.detail ?: error.problem.title ?: error.message.orEmpty()
            }
            is HttpException -> {
                val raw = peekHttpErrorBody(error)
                jsonObj = raw?.let { parseObject(it) }
                raw ?: error.message().orEmpty()
            }
            else -> error.message.orEmpty()
        }

        val code = (jsonObj?.stringValue("code") ?: jsonObj?.stringValue("error") ?: "")
            .trim()
            .lowercase()
        if (code.isNotEmpty()) return code

        val lower = rawText.lowercase()
        return when {
            lower.contains("reused_photo_evidence_refused") -> "reused_photo_evidence_refused"
            lower.contains("photo_evidence_required") -> "photo_evidence_required"
            lower.contains("invalid_evidence_uri") -> "invalid_evidence_uri"
            lower.contains("claim_window_expired") -> "claim_window_expired"
            lower.contains("claim_not_allowed") -> "claim_not_allowed"
            else -> null
        }
    }

    fun formatClaimErrorMessage(error: Throwable, fallback: String = "Claim failed"): String {
        return when (resolveClaimErrorCode(error)) {
            "reused_photo_evidence_refused" ->
                "This photo was already used for another claim. Please take a new photo."
            "photo_evidence_required" ->
                "Photo proof is required for this claim type."
            "invalid_evidence_uri" ->
                "Photo upload link is invalid. Please re-upload the photo."
            "claim_window_expired" ->
                "Claim window has expired (48h post-delivery)."
            else -> {
                when (error) {
                    is ProblemDetailException -> error.problem.detail ?: error.problem.title ?: fallback
                    is HttpException -> {
                        val raw = peekHttpErrorBody(error)
                        val obj = raw?.let { parseObject(it) }
                        obj?.stringValue("message")?.takeIf { it.isNotBlank() }
                            ?: obj?.stringValue("detail")?.takeIf { it.isNotBlank() }
                            ?: obj?.stringValue("error")?.takeIf { it.isNotBlank() }
                            ?: error.message()
                            ?: fallback
                    }
                    else -> error.message?.takeIf { it.isNotBlank() } ?: fallback
                }
            }
        }
    }

    private fun parseObject(raw: String): JsonObject? =
        runCatching { OrderLineContractHelper.lenientJson.parseToJsonElement(raw).jsonObject }.getOrNull()

    private fun JsonObject.stringValue(key: String): String? =
        runCatching { this[key]?.jsonPrimitive?.contentOrNull }.getOrNull()

    private fun JsonObject.longValue(key: String): Long? =
        runCatching { this[key]?.jsonPrimitive?.longOrNull }.getOrNull()
}
