package com.pegasusx.warehouse.util

data class OrderActionFlags(
    val canDelay: Boolean,
    val canReject: Boolean,
    val canOverflow: Boolean,
    val canReassign: Boolean,
)

fun orderActionFlags(state: String): OrderActionFlags {
    val s = state.uppercase()
    val terminal = s == "COMPLETED" || s == "CANCELLED"
    val inFlight = s == "LOADED" || s == "IN_TRANSIT"
    return OrderActionFlags(
        canDelay = !terminal && !inFlight,
        canReject = s in setOf(
            "PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED",
        ),
        canOverflow = s == "LOADED" || s == "IN_TRANSIT",
        canReassign = s in setOf(
            "PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED",
        ),
    )
}
