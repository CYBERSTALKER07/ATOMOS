package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.model.ArInvoiceDto
import com.pegasusx.retailer.data.model.ArInvoicesResponse
import com.pegasusx.retailer.data.model.CreditRelationshipDto
import com.pegasusx.retailer.data.model.CreditRelationshipsResponse
import com.pegasusx.retailer.data.model.ProblemDetailException
import com.pegasusx.retailer.data.model.RetailerJoinRequestConflict
import java.util.UUID
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException

data class QuarterlyPromiseLimitError(
    val code: String = "quarterly_promise_limit_reached",
    val quarterKey: String? = null,
    val message: String? = null,
)

object RetailerW3ContractHelper {

    fun generateIdempotencyUuid(): String = UUID.randomUUID().toString()

    fun parseCreditRelationshipsJson(element: JsonElement): List<CreditRelationshipDto> =
        runCatching {
            OrderLineContractHelper.lenientJson
                .decodeFromJsonElement<CreditRelationshipsResponse>(element)
                .relationships
        }.getOrElse { emptyList() }

    fun parseArInvoicesJson(element: JsonElement): List<ArInvoiceDto> =
        runCatching {
            OrderLineContractHelper.lenientJson
                .decodeFromJsonElement<ArInvoicesResponse>(element)
                .invoices
        }.getOrElse { emptyList() }

    fun resolveTermsDays(rel: CreditRelationshipDto): Int =
        if (rel.termsDays > 0) {
            rel.termsDays
        } else {
            rel.resolvedTerms?.termsDays ?: rel.termsDays
        }

    fun resolveGraceDays(rel: CreditRelationshipDto): Int =
        if (rel.gracePeriodDays > 0) {
            rel.gracePeriodDays
        } else {
            rel.resolvedTerms?.gracePeriodDays ?: rel.gracePeriodDays
        }

    fun undisputedMinor(inv: ArInvoiceDto): Long =
        inv.undisputedMinor ?: (inv.balanceMinor - inv.disputedMinor).coerceAtLeast(0L)

    fun sortArInvoicesOldestFirst(invoices: List<ArInvoiceDto>): List<ArInvoiceDto> =
        invoices.sortedWith(
            compareBy<ArInvoiceDto>(
                { sortableTimestampKey(it.dueAt) },
                { sortableTimestampKey(it.createdAt) },
                { it.invoiceId },
            ),
        )

    private val isoDatePrefixRegex =
        Regex("""^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2}):(\d{2}))?""")

    private fun sortableTimestampKey(raw: String?): String {
        val trimmed = raw?.trim().orEmpty()
        if (trimmed.isEmpty()) {
            return "9999-99-99T99:99:99"
        }
        val match = isoDatePrefixRegex.find(trimmed)
        if (match != null) {
            val year = match.groupValues[1]
            val month = match.groupValues[2]
            val day = match.groupValues[3]
            val hour = match.groupValues[4].ifEmpty { "00" }
            val min = match.groupValues[5].ifEmpty { "00" }
            val sec = match.groupValues[6].ifEmpty { "00" }
            return "$year-$month-${day}T$hour:$min:$sec"
        }
        return trimmed
    }

    fun parseQuarterlyPromiseLimitError(error: Throwable): QuarterlyPromiseLimitError? {
        if (error is ProblemDetailException) {
            val code = error.problem.code
            val raw = listOfNotNull(code, error.problem.detail, error.problem.title, error.message)
                .joinToString(" ")
            if (code == "quarterly_promise_limit_reached" ||
                raw.contains("quarterly_promise_limit_reached") ||
                error.problem.status == 409
            ) {
                return QuarterlyPromiseLimitError(
                    code = code ?: "quarterly_promise_limit_reached",
                    quarterKey = null,
                    message = error.problem.detail ?: error.problem.title ?: error.message,
                )
            }
            return null
        }

        if (error is HttpException) {
            val bodyStr = RetailerW2ContractHelper.peekHttpErrorBody(error)
            val parsedObj = bodyStr?.let { raw ->
                runCatching {
                    OrderLineContractHelper.lenientJson.parseToJsonElement(raw) as? JsonObject
                }.getOrNull()
            }
            val code = parsedObj?.get("code")?.jsonPrimitive?.contentOrNull
                ?: parsedObj?.get("error")?.jsonPrimitive?.contentOrNull
            val quarterKey = parsedObj?.get("quarter_key")?.jsonPrimitive?.contentOrNull
            val message = parsedObj?.get("message")?.jsonPrimitive?.contentOrNull
                ?: parsedObj?.get("detail")?.jsonPrimitive?.contentOrNull
                ?: bodyStr
            if (code == "quarterly_promise_limit_reached" ||
                (bodyStr?.contains("quarterly_promise_limit_reached") == true) ||
                error.code() == 409
            ) {
                return QuarterlyPromiseLimitError(
                    code = code ?: "quarterly_promise_limit_reached",
                    quarterKey = quarterKey,
                    message = message,
                )
            }
        }
        return null
    }

    fun parseArErrorMessage(error: Throwable, fallback: String): String {
        if (error is ProblemDetailException) {
            return error.problem.detail ?: error.problem.title ?: error.message ?: fallback
        }
        if (error is HttpException) {
            val bodyStr = RetailerW2ContractHelper.peekHttpErrorBody(error)
            val parsedObj = bodyStr?.let { raw ->
                runCatching {
                    OrderLineContractHelper.lenientJson.parseToJsonElement(raw) as? JsonObject
                }.getOrNull()
            }
            val msg = parsedObj?.get("message")?.jsonPrimitive?.contentOrNull
                ?: parsedObj?.get("detail")?.jsonPrimitive?.contentOrNull
                ?: parsedObj?.get("error")?.jsonPrimitive?.contentOrNull
            if (!msg.isNullOrBlank()) {
                return msg
            }
        }
        return error.message?.takeIf { it.isNotBlank() } ?: fallback
    }

    fun normalizedInnOrNull(raw: String): String? {
        val trimmed = raw.trim()
        return if (trimmed.length == 9 && trimmed.all { it in '0'..'9' }) trimmed else null
    }

    fun parseJoinRequestConflict(error: Throwable): RetailerJoinRequestConflict? {
        if (error is ProblemDetailException) {
            val raw = listOfNotNull(error.problem.code, error.problem.detail, error.problem.title, error.message)
                .joinToString(" ")
            if (error.problem.code == "join_request_pending" ||
                (error.problem.status == 409 && raw.contains("join_request_pending"))
            ) {
                return RetailerJoinRequestConflict(
                    error = "join_request_pending",
                    message = error.problem.detail ?: error.problem.title ?: error.message,
                )
            }
            return null
        }

        if (error is HttpException) {
            val bodyStr = RetailerW2ContractHelper.peekHttpErrorBody(error) ?: return null
            val parsedObj = runCatching {
                OrderLineContractHelper.lenientJson.parseToJsonElement(bodyStr) as? JsonObject
            }.getOrNull()
            val err = parsedObj?.get("error")?.jsonPrimitive?.contentOrNull
                ?: parsedObj?.get("code")?.jsonPrimitive?.contentOrNull
            if (err == "join_request_pending" || (error.code() == 409 && bodyStr.contains("join_request_pending"))) {
                val decoded = parsedObj?.let {
                    runCatching {
                        OrderLineContractHelper.lenientJson.decodeFromJsonElement<RetailerJoinRequestConflict>(it)
                    }.getOrNull()
                }
                return decoded ?: RetailerJoinRequestConflict(
                    error = "join_request_pending",
                    requestId = parsedObj?.get("request_id")?.jsonPrimitive?.contentOrNull,
                    targetTenantId = parsedObj?.get("target_tenant_id")?.jsonPrimitive?.contentOrNull,
                    legalName = parsedObj?.get("legal_name")?.jsonPrimitive?.contentOrNull,
                    normalizedInn = parsedObj?.get("normalized_inn")?.jsonPrimitive?.contentOrNull,
                    message = parsedObj?.get("message")?.jsonPrimitive?.contentOrNull ?: bodyStr,
                )
            }
        }
        return null
    }
}
