package com.pegasusx.retailer.data.model

import com.pegasus.design.MarketPackStore
import com.pegasus.design.packCurrency

private val FOREIGN_RAILS = setOf("STRIPE", "ADYEN", "AIRWALLEX")

fun selectableRetailerCatalogCodes(catalog: List<PSPListing>): List<String> {
    return catalog
        .filter { it.selectable }
        .map { it.code.trim().uppercase() }
        .filter { it.isNotEmpty() && it !in FOREIGN_RAILS }
}

/** Card rails the retailer may tap. Empty catalog never invents Adyen/Stripe. */
fun filterRetailerCardGateways(
    incoming: List<String>,
    catalogCodes: List<String>,
): List<String> {
    val allowed = catalogCodes.map { it.trim().uppercase() }.filter { it.isNotEmpty() && it != "CASH" }
    val raw = incoming.map { it.trim().uppercase() }.filter { it.isNotEmpty() }
    if (allowed.isNotEmpty()) {
        return if (raw.isEmpty()) allowed else raw.filter { it in allowed }
    }
    return raw.filter { it !in FOREIGN_RAILS }
}

fun sessionPackCurrency(): String = packCurrency(MarketPackStore.pack)

/** Stored/event currency, else session pack. Empty pack does not invent UZS. */
fun moneyCurrency(raw: String?): String {
    val fromEvent = raw?.trim().orEmpty()
    if (fromEvent.isNotEmpty()) return fromEvent.uppercase()
    return sessionPackCurrency()
}
