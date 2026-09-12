package com.pegasusx.driver.ui.screens.home

import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState

data class RemainingStop(
    val id: String,
    val title: String,
    val state: OrderState,
    val sequenceIndex: Int,
) {
    val firstClass: Boolean get() = RemainingStops.isFirstClass(state)
}

data class MoneyHealthCounts(
    val pendingCash: Int,
    val openFiscal: Int,
    val creditLeave: Int,
)

object RemainingStops {
    fun isFirstClass(state: OrderState): Boolean =
        state == OrderState.ARRIVED_SHOP_CLOSED || state == OrderState.FISCAL_FAILED

    fun remaining(orders: List<Order>): List<RemainingStop> =
        remainingStops(
            orders.map {
                RemainingStop(
                    id = it.id,
                    title = it.retailerName,
                    state = it.state,
                    sequenceIndex = it.sequenceIndex,
                )
            },
        )

    fun remainingStops(stops: List<RemainingStop>): List<RemainingStop> =
        stops
            .filter { it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED }
            .sortedWith(compareBy<RemainingStop> { it.sequenceIndex }.thenBy { it.id })

    fun moneyHealth(orders: List<Order>): MoneyHealthCounts =
        MoneyHealthCounts(
            pendingCash = orders.count {
                it.state == OrderState.PENDING_CASH_COLLECTION || it.state == OrderState.AWAITING_PAYMENT
            },
            openFiscal = orders.count {
                it.state == OrderState.FISCALIZING || it.state == OrderState.FISCAL_FAILED
            },
            creditLeave = orders.count { it.state == OrderState.DELIVERED_ON_CREDIT },
        )
}
