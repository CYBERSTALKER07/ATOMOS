package com.pegasusx.factory.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import com.pegasusx.factory.data.model.PulseEvent
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun HandoffTimelineSection(
    events: List<PulseEvent>,
    loading: Boolean,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        FactorySectionHeader(title = stringResource(R.string.warehouse_portal_dispatch_text_handoff_timeline), count = events.size)
        val subtitle = when {
            loading && events.isEmpty() -> "Loading handoff chain…"
            events.isEmpty() -> "No preorder → dispatch → seal events in the recent pulse window."
            else -> "${events.size} handoff event(s) in recent pulse."
        }
        Text(
            text = subtitle,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        events.take(8).forEach { event ->
            Card(
                colors = CardDefaults.elevatedCardColors(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(
                    modifier = Modifier.padding(PegasusSpacing.md),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
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
                            maxLines = 3,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                }
            }
        }
    }
}
