package com.pegasusx.retailer.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

/** Canonical demand source codes from L3 sell-through. */
const val SOURCE_STORE_POS = "STORE_POS"
const val SOURCE_WHOLESALE_HISTORY = "WHOLESALE_HISTORY"

fun normalizeDemandSources(sources: List<String>?): List<String> {
    if (!sources.isNullOrEmpty()) return sources
    return listOf(SOURCE_WHOLESALE_HISTORY)
}

fun demandSourceLabel(code: String): String = when (code.uppercase()) {
    SOURCE_STORE_POS -> "Store POS"
    SOURCE_WHOLESALE_HISTORY -> "Wholesale"
    else -> code
}

/**
 * Enterprise demand-source chips for reorder suggestions.
 * STORE_POS = floor sell-through; WHOLESALE_HISTORY = B2B sensing.
 */
@Composable
fun DemandSourceChips(
    sources: List<String>?,
    modifier: Modifier = Modifier,
) {
    val list = normalizeDemandSources(sources)
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        list.forEach { code ->
            val isPos = code.equals(SOURCE_STORE_POS, ignoreCase = true)
            val bg = if (isPos) {
                Color(0xFF16A34A).copy(alpha = 0.18f)
            } else {
                MaterialTheme.colorScheme.surfaceVariant
            }
            val fg = if (isPos) {
                Color(0xFF15803D)
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            }
            RetailerTagChip(
                text = demandSourceLabel(code).uppercase(),
                bgColor = bg,
                textColor = fg,
            )
        }
    }
}
