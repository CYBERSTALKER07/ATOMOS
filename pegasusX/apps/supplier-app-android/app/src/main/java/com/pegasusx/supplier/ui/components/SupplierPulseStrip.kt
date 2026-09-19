package com.pegasusx.supplier.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.PulseEvent
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.R

@Composable
fun SupplierPulseStrip(
    events: List<PulseEvent>,
    loading: Boolean,
    error: String? = null,
    modifier: Modifier = Modifier,
) {
    if (loading && events.isEmpty() && error.isNullOrBlank()) {
        Text(
            text = stringResource(R.string.mobile_supplier_ui_loading_network_pulse),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = modifier.padding(vertical = PegasusSpacing.sm),
        )
        return
    }
    if (!error.isNullOrBlank()) {
        Text(
            text = error,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.error,
            modifier = modifier.padding(vertical = PegasusSpacing.sm),
        )
        return
    }
    if (events.isEmpty()) return

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.factory_portal_app_text_network_pulse),
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(bottom = PegasusSpacing.sm),
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            events.take(12).forEach { event ->
                Card(
                    colors = CardDefaults.elevatedCardColors(),
                    modifier = Modifier.widthIn(min = 180.dp, max = 240.dp),
                ) {
                    Column(
                        modifier = Modifier.padding(PaddingValues(horizontal = 12.dp, vertical = 10.dp)),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        Text(
                            text = event.title,
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.SemiBold,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                        )
                        event.description?.takeIf { it.isNotBlank() }?.let { description ->
                            Text(
                                text = description,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                }
            }
        }
    }
}
