package com.pegasusx.warehouse.ui.components.orders

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.Order
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import java.text.NumberFormat
import java.util.Locale

fun LazyGridScope.orderLineItems(
    order: Order,
    fmt: NumberFormat = NumberFormat.getInstance(Locale("uz", "UZ"))
) {
    if (order.lineItems.isEmpty()) return

    item(span = { GridItemSpan(maxLineSpan) }) {
        HorizontalDivider()
        Spacer(Modifier.height(PegasusSpacing.sm))
        Text("Line Items", style = MaterialTheme.typography.titleMedium)
    }
    
    items(order.lineItems) { item ->
        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
            Row(
                modifier = Modifier.padding(PegasusSpacing.lg),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        item.productName.ifBlank { "Product" },
                        style = MaterialTheme.typography.titleSmall
                    )
                    Text(
                        "Qty: ${item.quantity}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Text(
                    "${fmt.format(item.unitPrice)} UZS",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
