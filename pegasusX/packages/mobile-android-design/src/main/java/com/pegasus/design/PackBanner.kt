package com.pegasus.design

import androidx.compose.foundation.border
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp

/** GS-R login splash: currency + receipts. */
@Composable
fun PackBanner(pack: MarketPack?, modifier: Modifier = Modifier) {
    if (pack == null || pack.currencyCode.isBlank()) return
    Text(
        text = "${pack.currencyCode} · receipts: ${pack.receiptLabel}",
        style = MaterialTheme.typography.labelMedium,
        modifier = modifier
            .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(999.dp))
            .padding(horizontal = 10.dp, vertical = 4.dp)
            .testTag("gs-r-pack-chip"),
    )
}
