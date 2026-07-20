package com.pegasusx.driver.ui.screens.map

import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState

/**
 * Map execution phases — mirrors iOS FleetMapView flow mapped to active-order states.
 */
enum class MapPhase {
    IDLE,
    NAVIGATING,
    ARRIVED,
    VERIFYING,
}

fun resolveActiveOrder(orders: List<Order>): Order? {
    val executionStates = setOf(
        OrderState.IN_TRANSIT,
        OrderState.ARRIVING,
        OrderState.ARRIVED,
        OrderState.ARRIVED_SHOP_CLOSED,
        OrderState.AWAITING_PAYMENT,
        OrderState.PENDING_CASH_COLLECTION,
        OrderState.FISCALIZING,
        OrderState.FISCAL_FAILED,
        OrderState.DISPATCHED,
        OrderState.LOADED,
    )
    return orders.firstOrNull { order ->
        order.state in executionStates &&
            order.latitude != null &&
            order.longitude != null
    } ?: orders.firstOrNull { order ->
        order.state != OrderState.COMPLETED &&
            order.state != OrderState.CANCELLED &&
            order.latitude != null &&
            order.longitude != null
    }
}

fun resolveMapPhase(activeOrder: Order?): MapPhase {
    if (activeOrder == null) return MapPhase.IDLE
    return when (activeOrder.state) {
        OrderState.IN_TRANSIT,
        OrderState.DISPATCHED,
        OrderState.LOADED,
        OrderState.ARRIVING,
        -> MapPhase.NAVIGATING

        OrderState.ARRIVED,
        OrderState.ARRIVED_SHOP_CLOSED,
        -> MapPhase.ARRIVED

        OrderState.AWAITING_PAYMENT,
        OrderState.PENDING_CASH_COLLECTION,
        OrderState.FISCALIZING,
        OrderState.FISCAL_FAILED,
        -> MapPhase.VERIFYING

        else -> MapPhase.IDLE
    }
}
