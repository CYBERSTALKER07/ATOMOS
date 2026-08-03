package com.pegasusx.warehouse.ui.screens.staff

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.StaffMember
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

@Composable
fun StaffList(
    staff: List<StaffMember>,
<<<<<<< HEAD
    modifier: Modifier = Modifier
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = modifier,
    ) {
        items(staff, key = { it.workerId }) { s ->
            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(s.name, style = MaterialTheme.typography.titleSmall)
                        Text(
                            "${s.role} · ${s.phone}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    AssistChip(
                        onClick = {},
                        label = { Text(if (s.isActive) "Active" else "Inactive", style = MaterialTheme.typography.labelSmall) },
                        colors = if (s.isActive) AssistChipDefaults.assistChipColors()
                        else AssistChipDefaults.assistChipColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                    )
=======
    loading: Boolean,
    error: String?,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier
) {
    when {
        loading && staff.isEmpty() -> Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        error != null && staff.isEmpty() -> Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(error, color = MaterialTheme.colorScheme.error)
                Spacer(Modifier.height(PegasusSpacing.lg))
                Button(onClick = onRetry) { Text("Retry") }
            }
        }
        staff.isEmpty() -> Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            Text("No staff members", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        else -> LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            modifier = modifier.fillMaxSize(),
        ) {
            items(staff, key = { it.workerId }) { s ->
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Row(modifier = Modifier.padding(PegasusSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(s.name, style = MaterialTheme.typography.titleSmall)
                            Text(
                                "${s.role} · ${s.phone}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        AssistChip(
                            onClick = {},
                            label = { Text(if (s.isActive) "Active" else "Inactive", style = MaterialTheme.typography.labelSmall) },
                            colors = if (s.isActive) AssistChipDefaults.assistChipColors()
                            else AssistChipDefaults.assistChipColors(containerColor = MaterialTheme.colorScheme.errorContainer),
                        )
                    }
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
                }
            }
        }
    }
}
