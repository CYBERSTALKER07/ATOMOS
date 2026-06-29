package com.pegasusx.driver.ui.navigation

import java.net.URI

/** Parsed native destination for a portal-style handoff `primary_link`. */
sealed class HandoffDestination {
    data object Home : HandoffDestination()
    data object FleetMap : HandoffDestination()
    data object ManifestList : HandoffDestination()
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
            "manifests" -> when {
                segments.size >= 2 -> HandoffDestination.ManifestDetail(segments[1])
                else -> HandoffDestination.ManifestList
            }
            "dispatch" -> HandoffDestination.Home
            "fleet" -> HandoffDestination.FleetMap
            "orders" -> when {
                segments.size >= 2 -> HandoffDestination.OrderDetail(segments[1])
                else -> HandoffDestination.Unresolved
            }
            else -> HandoffDestination.Unresolved
        }
    }
}
