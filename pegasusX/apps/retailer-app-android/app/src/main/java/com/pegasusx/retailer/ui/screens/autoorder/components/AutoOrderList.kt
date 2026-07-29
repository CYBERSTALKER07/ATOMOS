package com.pegasusx.retailer.ui.screens.autoorder.components

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.theme.SoftSquircleShape

@Composable
fun ForecastRow(forecast: DemandForecast) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Confidence ring
            Box(contentAlignment = Alignment.Center) {
                CircularProgressIndicator(
                    progress = { forecast.confidence.toFloat() },
                    modifier = Modifier.size(36.dp),
                    strokeWidth = 2.5.dp,
                    trackColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f),
                    color = when {
                        forecast.confidence >= 0.8 -> Color(0xFF4CAF50)
                        forecast.confidence >= 0.6 -> Color(0xFFFF9800)
                        else -> Color(0xFFF44336)
                    },
                )
                Text(
                    forecast.confidencePercent,
                    style = MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                )
            }
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(forecast.productName, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium), maxLines = 1)
                Text(
                    "Order by ${forecast.suggestedOrderDate}",
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 10.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                )
            }
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text("${forecast.predictedQuantity}", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.primary)
                Text("units", style = MaterialTheme.typography.labelSmall.copy(fontSize = 8.sp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
            }
        }
    }
}
