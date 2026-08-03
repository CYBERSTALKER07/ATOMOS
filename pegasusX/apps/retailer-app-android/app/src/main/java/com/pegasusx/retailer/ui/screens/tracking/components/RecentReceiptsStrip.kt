package com.pegasusx.retailer.ui.screens.tracking.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.ui.components.RetailerListCard

@Composable
fun RecentReceiptsStrip(
    receipts: List<TrackingOrder>,
    modifier: Modifier = Modifier,
) {
    var fiscalQrOrder by remember { mutableStateOf<TrackingOrder?>(null) }

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            "Recent receipts",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            "Completed deliveries from the tracking feed. Tap for fiscal QR when available.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 2.dp, bottom = 8.dp),
        )
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(max = 160.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            receipts.forEach { receipt ->
                val hasFiscalQr = receipt.fiscalQr.isNotBlank()
                RetailerListCard(
                    headline = receipt.supplierName.ifBlank { "Supplier" },
                    supporting = buildString {
                        append("#${receipt.orderId.takeLast(8)}")
                        append(" · ")
                        append(receipt.fiscalReceiptLabel)
                        append(" · ")
                        append(String.format("%,d", receipt.totalAmount))
                        if (hasFiscalQr) append(" · Fiscal QR")
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .then(
                            if (hasFiscalQr) {
                                Modifier.clickable { fiscalQrOrder = receipt }
                            } else {
                                Modifier
                            },
                        ),
                )
            }
        }
    }

    FiscalReceiptQROverlay(
        order = fiscalQrOrder,
        onDismiss = { fiscalQrOrder = null },
    )
}
