package com.pegasusx.warehouse.util

import com.pegasusx.warehouse.data.model.PulseEvent

private val HANDOFF_KINDS = setOf(
    "PREORDER",
    "ORDER_ACCEPTED",
    "ORDER_DISPATCHED",
    "MANIFEST_SEALED",
    "MANIFEST_DISPATCHED",
    "DISPATCH",
)

fun filterHandoffPulseEvents(events: List<PulseEvent>): List<PulseEvent> =
    events.filter(::isHandoffPulseEvent)

private fun isHandoffPulseEvent(event: PulseEvent): Boolean {
    val haystack = "${event.kind} ${event.title}".uppercase()
    if (event.kind.uppercase() in HANDOFF_KINDS) return true
    return Regex("PREORDER|ACCEPT|DISPATCH|SEAL|MANIFEST").containsMatchIn(haystack)
}
