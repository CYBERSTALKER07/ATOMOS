package com.pegasusx.mobilekit.offline

/**
 * Role apps implement eligibility; kit provides storage + HTTP outcome helpers.
 */
interface OfflineEndpointCatalog {
    fun isOfflineEligible(endpoint: String): Boolean
    fun priorityFor(endpoint: String): Int = 40
    fun normalize(endpoint: String): String = OfflineHttpSemantics.normalizeEndpoint(endpoint)
}
