package com.pegasusx.warehouse.util

data class OrderActionFlags(
    val canDelay: Boolean,
    val canReject: Boolean,
    val canOverflow: Boolean,
)

fun orderActionFlags(state: String): OrderActionFlags {
    val s = state.uppercase()
    return OrderActionFlags(
        canDelay = s == "PENDING" || s == "LOADED",
        canReject = s == "PENDING" || s == "LOADED" || s == "IN_TRANSIT" || s == "SCHEDULED" || s == "AUTO_ACCEPTED",
        canOverflow = s == "LOADED" || s == "IN_TRANSIT",
    )
}
