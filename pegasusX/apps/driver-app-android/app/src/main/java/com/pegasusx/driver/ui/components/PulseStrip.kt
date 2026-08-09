package com.pegasusx.driver.ui.components

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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.model.PulseEvent
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.R

@Composable
fun PulseStrip(
    events: List<PulseEvent>,
    loading: Boolean,
    modifier: Modifier = Modifier,
) {
    if (loading && events.isEmpty()) {
        Text(
            text = stringResource(R.string.mobile_driver_ui_loading_network_pulse),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = modifier.padding(vertical = PegasusSpacing.s8),
        )
        return
    }
    if (events.isEmpty()) return

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.factory_portal_app_text_network_pulse),
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(bottom = PegasusSpacing.s8),
        )
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.s12),
        ) {
            events.take(12).forEach { event ->
                PegasusCard(
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
