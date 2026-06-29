package com.pegasus.payload.ui.navigation

import java.net.URI

sealed class HandoffDestination {
    data object TruckList : HandoffDestination()
    data class ManifestDetail(val manifestId: String) : HandoffDestination()
    data class OrderDetail(val orderId: String) : HandoffDestination()
    data object Unresolved : HandoffDestination()
}

object HandoffPathResolver {
    fun resolve(link: String): HandoffDestination {
        val trimmed = link.trim()
        if (trimmed.isBlank()) return HandoffDestination.Unresolved

        val path = runCatching {
            if (trimmed.startsWith("/")) trimmed else URI(trimmed).path
        }.getOrDefault(trimmed).substringBefore('?').trimEnd('/')

        val segments = path.split('/').filter { it.isNotBlank() }
        if (segments.isEmpty()) return HandoffDestination.Unresolved

        return when (segments.first()) {
            "manifests", "dispatch" -> when {
                segments.size >= 2 -> HandoffDestination.ManifestDetail(segments[1])
                else -> HandoffDestination.TruckList
            }
            "fleet" -> HandoffDestination.TruckList
            "orders" -> when {
                segments.size >= 2 -> HandoffDestination.OrderDetail(segments[1])
                else -> HandoffDestination.Unresolved
            }
            else -> HandoffDestination.Unresolved
        }
    }
}
