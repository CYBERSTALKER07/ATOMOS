package com.pegasus.payload.ui.home

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasus.payload.data.model.ManifestExceptionRow
import com.pegasus.payload.ui.components.PayloadSpacing
import com.pegasus.payload.ui.components.PayloadStatusChip
import com.pegasus.payload.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestExceptionsSheet(
    items: List<ManifestExceptionRow>,
    loading: Boolean,
    onDismiss: () -> Unit,
    onRefresh: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text("Manifest exceptions", style = MaterialTheme.typography.titleLarge)
                IconButton(onClick = onRefresh) {
                    Icon(Icons.Filled.Refresh, contentDescription = stringResource(R.string.mobile_payload_ui_refresh_exceptions))
                }
            }
            
            // --- Parity: Report Dock Damage ---
            androidx.compose.foundation.layout.Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 8.dp)
                    .androidx.compose.foundation.background(androidx.compose.ui.graphics.Color(0xFFFEE2E2), shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp))
                    .padding(16.dp)
            ) {
                Column {
                    Text("Report Dock Damage", color = androidx.compose.ui.graphics.Color(0xFF991B1B), fontWeight = androidx.compose.ui.text.font.FontWeight.Bold)
                    Spacer(Modifier.height(8.dp))
                    androidx.compose.material3.Button(
                        onClick = { /* ViewModel logic wired later */ },
                        colors = androidx.compose.material3.ButtonDefaults.buttonColors(containerColor = androidx.compose.ui.graphics.Color(0xFFDC2626)),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("MARK DAMAGED", color = androidx.compose.ui.graphics.Color.White, fontWeight = androidx.compose.ui.text.font.FontWeight.Bold)
                    }
                }
            }
            // ----------------------------------
            
            HorizontalDivider()
            when {
                loading && items.isEmpty() -> com.pegasus.design.PegasusLoadingState(
                    title = stringResource(R.string.mobile_payload_ui_loading_exceptions),
                    body = "Fetching overflow, damaged, and manual removals.",
                    modifier = Modifier.fillMaxWidth().heightIn(min = 200.dp),
                )
                items.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No exceptions",
                    body = "Overflow, damaged, and manual removals appear here.",
                    modifier = Modifier.fillMaxWidth().heightIn(min = 200.dp),
                )
                else -> LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    Modifier.fillMaxWidth()
                ) {
                    items(items, key = { it.exceptionId }) { row ->
                        Column(
                            Modifier
                                .fillMaxWidth()
                                .padding(vertical = PayloadSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(PayloadSpacing.xs),
                        ) {
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(PayloadSpacing.sm),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                PayloadStatusChip(status = row.reason)
                                if (row.escalated) {
                                    PayloadStatusChip(status = "ESCALATED")
                                }
                            }
                            Text(
                                stringResource(R.string.mobile_payload_ui_order_take_manifest_take_2, row.orderId.take(8), row.manifestId.take(8)),
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            Text(
                                stringResource(R.string.mobile_payload_ui_attempts_attemptcount, row.attemptCount),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        HorizontalDivider()
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
        }
    }
}
