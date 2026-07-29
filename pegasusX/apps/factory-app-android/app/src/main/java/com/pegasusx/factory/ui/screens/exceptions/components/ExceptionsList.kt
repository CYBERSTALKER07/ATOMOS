package com.pegasusx.factory.ui.screens.exceptions.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.data.model.ManifestException
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun ExceptionsList(
    exceptions: List<ManifestException>,
    escalatedOnly: Boolean,
    onEscalatedOnlyChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = modifier.fillMaxSize(),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                FilterChip(
                    selected = !escalatedOnly,
                    onClick = { onEscalatedOnlyChange(false) },
                    label = { Text("All") },
                )
                FilterChip(
                    selected = escalatedOnly,
                    onClick = { onEscalatedOnlyChange(true) },
                    label = { Text("Escalated only") },
                )
            }
        }
        if (exceptions.isEmpty()) {
            item {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = if (escalatedOnly) "No escalated exceptions" else "No exceptions",
                    body = if (escalatedOnly) {
                        "No transfers have hit the DLQ threshold (3+ overflows)."
                    } else {
                        "All manifest loading operations completed without exceptions."
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        } else {
            items(exceptions, key = { it.exceptionId }) { exception ->
                ExceptionCard(exception = exception)
            }
        }
    }
}
