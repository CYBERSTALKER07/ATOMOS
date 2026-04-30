package com.thelab.retailer.ui.components

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import kotlinx.coroutines.delay
import java.time.Instant
import java.time.format.DateTimeParseException

@Composable
fun CountdownTimer(
    targetIso: String?,
    modifier: Modifier = Modifier,
    style: TextStyle = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold),
    color: Color = MaterialTheme.colorScheme.onSurface,
) {
    val targetEpoch = remember(targetIso) {
        targetIso?.let {
            try {
                Instant.parse(it).epochSecond
            } catch (_: DateTimeParseException) {
                null
            }
        }
    }

    var remaining by remember { mutableLongStateOf(0L) }

    LaunchedEffect(targetEpoch) {
        if (targetEpoch == null) return@LaunchedEffect
        while (true) {
            val now = Instant.now().epochSecond
            remaining = (targetEpoch - now).coerceAtLeast(0)
            if (remaining <= 0) break
            delay(1000)
        }
    }

    if (targetEpoch != null && remaining > 0) {
        val h = remaining / 3600
        val m = (remaining % 3600) / 60
        val s = remaining % 60
        Text(
            text = String.format("%02d:%02d:%02d", h, m, s),
            style = style,
            color = color,
            modifier = modifier,
        )
    }
}
