package com.pegasus.payload.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.StatusExplain

@Composable
fun ExplainStatusBanner(
    explain: StatusExplain?,
    fallbackTitle: String? = null,
    fallbackDetail: String? = null,
    modifier: Modifier = Modifier,
) {
    val title = explain?.title?.takeIf { it.isNotBlank() } ?: fallbackTitle ?: return
    val summary = explain?.summary?.takeIf { it.isNotBlank() } ?: fallbackDetail
    Surface(
        color = MaterialTheme.colorScheme.errorContainer,
        contentColor = MaterialTheme.colorScheme.onErrorContainer,
        shape = RoundedCornerShape(12.dp),
        modifier = modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
            summary?.let {
                Text(it, style = MaterialTheme.typography.bodySmall)
            }
            explain?.nextSteps?.takeIf { it.isNotEmpty() }?.forEach { step ->
                Text(stringResource(R.string.mobile_payload_ui_step, step), style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}
