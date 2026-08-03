package com.pegasusx.factory.ui.screens.transfer.components

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.theme.PegasusSpacing

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferFilters(
    filters: List<String>,
    selectedFilter: String,
    onFilterSelected: (String) -> Unit
) {
    Row(
        modifier = Modifier
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        filters.forEach { filter ->
            FilterChip(
                selected = selectedFilter == filter,
                onClick = { onFilterSelected(filter) },
                label = { Text(filter, style = MaterialTheme.typography.labelSmall) },
            )
        }
    }
}
