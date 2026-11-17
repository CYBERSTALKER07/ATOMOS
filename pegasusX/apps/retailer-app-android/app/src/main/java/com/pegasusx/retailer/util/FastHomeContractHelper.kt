package com.pegasusx.retailer.util

import com.pegasusx.retailer.data.local.RetailerLanguage
import com.pegasusx.retailer.data.local.RetailerWorkspaceMode
import com.pegasusx.retailer.data.model.ControlTowerPulse
import com.pegasusx.retailer.data.model.CreditProfile
import com.pegasusx.retailer.data.model.FastHomePendingProposal
import com.pegasusx.retailer.data.model.FastHomeProposalItem
import com.pegasusx.retailer.data.model.FastHomeTomorrowDelivery
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderStatus
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasusx.retailer.data.model.RetailerFastHomeResponse
import java.time.Instant
import java.time.LocalDateTime
import java.time.OffsetDateTime
import java.time.ZoneOffset
import java.time.ZonedDateTime
import kotlin.math.max
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.longOrNull

data class CutoffCountdownState(
    val valid: Boolean = false,
    val expired: Boolean = false,
    val remainingMs: Long = 0L,
    val formatted: String = "--:--:--",
)

object FastHomeContractHelper {

    fun parseServerTimestampMillis(raw: String?): Long? {
        val text = raw?.trim().orEmpty()
        if (text.isEmpty()) return null
        return runCatching { Instant.parse(text).toEpochMilli() }.getOrNull()
            ?: runCatching { OffsetDateTime.parse(text).toInstant().toEpochMilli() }.getOrNull()
            ?: runCatching { ZonedDateTime.parse(text).toInstant().toEpochMilli() }.getOrNull()
            ?: runCatching { LocalDateTime.parse(text).toInstant(ZoneOffset.UTC).toEpochMilli() }.getOrNull()
    }

    /**
     * Computes `serverNowMs - clientNowMs` so all subsequent countdown ticks stay
     * immune to device clock drift.
     */
    fun computeServerClockOffsetMs(
        serverNowIso: String?,
        clientNowMs: Long = System.currentTimeMillis(),
    ): Long {
        val serverMs = parseServerTimestampMillis(serverNowIso) ?: return 0L
        return serverMs - clientNowMs
    }

    fun computeCutoffCountdown(
        cutoffAtIso: String?,
        serverClockOffsetMs: Long,
        clientNowMs: Long = System.currentTimeMillis(),
    ): CutoffCountdownState {
        val targetMs = parseServerTimestampMillis(cutoffAtIso)
            ?: return CutoffCountdownState(valid = false, expired = false, remainingMs = 0L, formatted = "--:--:--")
        val adjustedNowMs = clientNowMs + serverClockOffsetMs
        val remainingMs = max(0L, targetMs - adjustedNowMs)
        val totalSeconds = remainingMs / 1000L
        val hours = totalSeconds / 3600L
        val minutes = (totalSeconds % 3600L) / 60L
        val seconds = totalSeconds % 60L
        return CutoffCountdownState(
            valid = true,
            expired = remainingMs <= 0L,
            remainingMs = remainingMs,
            formatted = "%02d:%02d:%02d".format(hours, minutes, seconds),
        )
    }

    /**
     * Owners default to Full ERP; Buyers, Cashiers, and Managers default to Fast Mode.
     */
    fun resolveDefaultWorkspaceMode(
        retailerRole: String?,
        isOwner: Boolean = false,
    ): RetailerWorkspaceMode {
        val normalized = retailerRole?.trim()?.uppercase().orEmpty()
        return if (isOwner || normalized == "OWNER") {
            RetailerWorkspaceMode.FULL
        } else {
            RetailerWorkspaceMode.FAST
        }
    }

    fun resolveEffectiveWorkspaceMode(
        persistedMode: RetailerWorkspaceMode?,
        retailerRole: String?,
        isOwner: Boolean = false,
    ): RetailerWorkspaceMode {
        return persistedMode ?: resolveDefaultWorkspaceMode(retailerRole, isOwner)
    }

    fun buildFastHomeFallback(
        orders: List<Order> = emptyList(),
        predictions: List<RetailerAIPrediction> = emptyList(),
        creditProfile: CreditProfile? = null,
        creditRelationshipsJson: JsonElement? = null,
        pulse: ControlTowerPulse? = null,
        pendingApprovalsCount: Int = 0,
        nowMs: Long = System.currentTimeMillis(),
        locale: RetailerLanguage = RetailerLanguage.UZ,
        currency: String = "UZS",
    ): RetailerFastHomeResponse {
        val effectiveCurrency = currency.ifBlank {
            orders.firstOrNull { it.currency.isNotBlank() }?.currency
                ?: predictions.firstOrNull { it.currency.isNotBlank() }?.currency
                ?: "UZS"
        }

        val activeDeliveries = orders
            .filter {
                it.status.isActive ||
                    it.status == OrderStatus.PENDING ||
                    it.status == OrderStatus.PENDING_REVIEW ||
                    it.status == OrderStatus.SCHEDULED
            }
            .sortedBy { order ->
                parseServerTimestampMillis(
                    order.proposedDeliveryDate ?: order.estimatedDelivery ?: order.deliverBefore ?: order.createdAt,
                ) ?: Long.MAX_VALUE
            }

        val tomorrowOrder = activeDeliveries.firstOrNull()
        val tomorrowDelivery = tomorrowOrder?.let { order ->
            FastHomeTomorrowDelivery(
                orderId = order.id,
                supplierId = order.supplierId,
                supplierName = order.supplierName,
                state = order.status.name,
                itemCount = order.itemCount,
                amountMinor = order.totalAmount,
                currency = order.currency.ifBlank { effectiveCurrency },
                deliveryDate = order.proposedDeliveryDate ?: order.estimatedDelivery ?: order.deliverBefore,
            )
        }

        val proposalOrder = orders.firstOrNull { order ->
            order.needsAiConfirmation || order.needsPreorderAction || order.needsDeliveryProposalReview
        }

        val pendingProposal: FastHomePendingProposal? = when {
            proposalOrder != null -> {
                val items = proposalOrder.items.map { item ->
                    val unitMinor = item.unitPrice.toLong()
                    FastHomeProposalItem(
                        productId = item.productId.ifBlank { item.id },
                        sku = item.productId.ifBlank { item.id },
                        productName = item.productName.ifBlank { item.productId },
                        quantity = item.quantity,
                        unitPriceMinor = unitMinor,
                        lineTotalMinor = unitMinor * item.quantity,
                    )
                }
                FastHomePendingProposal(
                    orderId = proposalOrder.id,
                    source = proposalOrder.orderSource.ifBlank { "MANUAL_PREORDER" },
                    supplierId = proposalOrder.supplierId,
                    supplierName = proposalOrder.supplierName,
                    state = proposalOrder.status.name,
                    confirmationStatus = proposalOrder.confirmationStatus,
                    proposedDeliveryDate = proposalOrder.proposedDeliveryDate ?: proposalOrder.deliverBefore,
                    autoConfirmAt = proposalOrder.autoConfirmAt,
                    itemCount = proposalOrder.itemCount,
                    amountMinor = proposalOrder.totalAmount,
                    currency = proposalOrder.currency.ifBlank { effectiveCurrency },
                    items = items,
                )
            }
            predictions.isNotEmpty() -> {
                val pred = predictions.first()
                val items = pred.lineItems.map { item ->
                    FastHomeProposalItem(
                        productId = item.sku,
                        sku = item.sku,
                        productName = item.name.ifBlank { item.sku },
                        quantity = item.quantity.toInt(),
                        unitPriceMinor = item.unitPriceMinor,
                        lineTotalMinor = item.unitPriceMinor * item.quantity,
                    )
                }
                FastHomePendingProposal(
                    orderId = pred.orderId,
                    source = pred.orderSource.ifBlank { "AI_PREDICTED" },
                    supplierId = pred.supplierId,
                    state = pred.confirmationStatus.ifBlank { "PENDING_REVIEW" },
                    confirmationStatus = pred.confirmationStatus,
                    proposedDeliveryDate = pred.requestedDeliveryDate,
                    autoConfirmAt = pred.autoConfirmAt,
                    itemCount = pred.quantity.toInt(),
                    amountMinor = pred.totalMinor,
                    currency = pred.currency.ifBlank { effectiveCurrency },
                    items = items,
                )
            }
            else -> null
        }

        val relItems = runCatching {
            val obj = creditRelationshipsJson as? JsonObject
            when {
                obj?.get("relationships") != null -> obj["relationships"]!!.jsonArray
                obj?.get("items") != null -> obj["items"]!!.jsonArray
                creditRelationshipsJson != null -> creditRelationshipsJson.jsonArray
                else -> emptyList()
            }
        }.getOrDefault(emptyList())

        var relBalance = 0L
        var relHeadroom = 0L
        var hasHold = false
        for (entry in relItems) {
            val obj = entry as? JsonObject ?: continue
            relBalance += (obj["current_balance_minor"] as? JsonPrimitive)?.longOrNull ?: 0L
            relHeadroom += (obj["available_credit_minor"] as? JsonPrimitive)?.longOrNull ?: 0L
            val onHold = (obj["on_hold"] as? JsonPrimitive)?.booleanOrNull == true
            val profileStatus = (obj["profile_status"] as? JsonPrimitive)?.contentOrNull.orEmpty()
            if (onHold || profileStatus.equals("ON_HOLD", ignoreCase = true)) {
                hasHold = true
            }
        }

        val balanceOwedMinor = creditProfile?.currentBalanceMinor ?: relBalance
        val creditHeadroomMinor = creditProfile?.availableCreditMinor ?: relHeadroom
        val delinquencyCount = (creditProfile?.delinquencyCount ?: 0L).toInt().coerceAtLeast(0)
        val overdueMinor = if (hasHold) balanceOwedMinor else 0L

        val nowInstant = Instant.ofEpochMilli(nowMs)
        val defaultCutoffIso = computeDefaultCutoffIso(nowInstant)

        return RetailerFastHomeResponse(
            serverNow = nowInstant.toString(),
            cutoffAt = pendingProposal?.autoConfirmAt?.takeIf { it.isNotBlank() } ?: defaultCutoffIso,
            nextDeliveryDate = tomorrowDelivery?.deliveryDate ?: pendingProposal?.proposedDeliveryDate,
            tomorrowDelivery = tomorrowDelivery,
            balanceOwedMinor = balanceOwedMinor,
            overdueMinor = overdueMinor,
            daysOverdue = delinquencyCount,
            creditHeadroomMinor = creditHeadroomMinor,
            pendingProposal = pendingProposal,
            pendingApprovalsCount = if (pendingApprovalsCount > 0) pendingApprovalsCount else 0,
            locale = locale.code,
            currency = effectiveCurrency,
            degradedFallback = true,
        )
    }

    private fun computeDefaultCutoffIso(nowInstant: Instant): String {
        // Default 18:00 Tashkent (13:00 UTC) daily ordering cutoff when server proposal has no explicit deadline.
        val utcNow = nowInstant.atOffset(ZoneOffset.UTC)
        var cutoff = utcNow.withHour(13).withMinute(0).withSecond(0).withNano(0)
        if (!cutoff.isAfter(utcNow)) {
            cutoff = cutoff.plusDays(1)
        }
        return cutoff.toInstant().toString()
    }
}
