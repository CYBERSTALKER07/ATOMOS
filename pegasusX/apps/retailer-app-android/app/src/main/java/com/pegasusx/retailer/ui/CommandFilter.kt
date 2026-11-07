package com.pegasusx.retailer.ui

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import com.pegasus.design.canonicalizeOrderStatus
import com.pegasusx.retailer.data.model.Order

class CommandFilterState {
    var orderStatus by mutableStateOf<String?>(null)
    var supplierId by mutableStateOf<String?>(null)

    fun jump(status: String, supplierId: String? = null) {
        this.orderStatus = canonicalizeOrderStatus(status)
        this.supplierId = supplierId
    }

    fun clear() {
        orderStatus = null
        supplierId = null
    }
}

val LocalCommandFilter = staticCompositionLocalOf { CommandFilterState() }

fun retailerOrderMatchesCommand(
    statusRaw: String,
    supplierId: String?,
    commandStatus: String,
    commandSupplierId: String?,
): Boolean {
    if (canonicalizeOrderStatus(statusRaw) != canonicalizeOrderStatus(commandStatus)) {
        return false
    }
    if (!commandSupplierId.isNullOrBlank() && supplierId != commandSupplierId) {
        return false
    }
    return true
}

fun List<Order>.filterCommand(status: String?, supplierId: String?): List<Order> {
    if (status.isNullOrBlank() && supplierId.isNullOrBlank()) return this
    return filter { order ->
        if (!status.isNullOrBlank() &&
            !retailerOrderMatchesCommand(order.status.name, order.supplierId, status, supplierId)
        ) {
            return@filter false
        }
        if (!supplierId.isNullOrBlank() && order.supplierId != supplierId) {
            return@filter false
        }
        true
    }
}
