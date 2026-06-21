package com.pegasusx.retailer.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.api.PegasusApiEntryPoint
import com.pegasusx.retailer.data.model.OrderTimelineEntry
import dagger.hilt.android.EntryPointAccessors
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
fun OrderStatusHistorySection(
    orderId: String,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val api = remember {
        EntryPointAccessors.fromApplication(
            context.applicationContext,
            PegasusApiEntryPoint::class.java,
        ).pegasusApi()
    }
    var items by remember(orderId) { mutableStateOf<List<OrderTimelineEntry>>(emptyList()) }
    var loading by remember(orderId) { mutableStateOf(true) }

    LaunchedEffect(orderId, api) {
        loading = true
        items = try {
            api.getOrderTimeline(orderId).items
        } catch (_: Exception) {
            emptyList()
        }
        loading = false
    }

    Column(modifier = modifier.fillMaxWidth()) {
        Text(
            "Status history",
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
        )
        Spacer(modifier = Modifier.height(8.dp))
        when {
            loading -> Text(
                "Loading…",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            )
            items.isEmpty() -> Text(
                "No status history yet.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            )
            else -> items.forEach { entry ->
                Column(modifier = Modifier.padding(vertical = 6.dp)) {
                    val previous = entry.previousStatus.orEmpty()
                    val label = if (previous.isNotBlank()) {
                        "$previous → ${entry.newStatus}"
                    } else {
                        entry.newStatus
                    }
                    Text(label, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
                    Text(
                        buildString {
                            append(formatWhen(entry.createdAt))
                            entry.reason?.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                    )
                }
            }
        }
    }
}

private fun formatWhen(iso: String): String {
    return try {
        val dt = Instant.parse(iso).atZone(ZoneId.systemDefault())
        DateTimeFormatter.ofPattern("MMM d, yyyy HH:mm").format(dt)
    } catch (_: Exception) {
        iso
    }
}
