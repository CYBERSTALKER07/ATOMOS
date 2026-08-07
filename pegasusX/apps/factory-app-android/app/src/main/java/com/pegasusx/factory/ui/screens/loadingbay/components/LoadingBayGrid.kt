package com.pegasusx.factory.ui.screens.loadingbay.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.data.model.PulseEvent
import com.pegasusx.factory.data.model.Transfer
import com.pegasusx.factory.ui.components.FactorySectionHeader
import com.pegasusx.factory.ui.components.HandoffTimelineSection
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun LoadingBayGrid(
    approved: List<Transfer>,
    loadingState: List<Transfer>,
    dispatched: List<Transfer>,
    handoffEvents: List<PulseEvent>,
    handoffLoading: Boolean,
    onTransferClick: (String) -> Unit,
    innerPadding: PaddingValues
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
        item {
            BayOverviewCard(
                readyCount = approved.size,
                loadingCount = loadingState.size,
                dispatchedCount = dispatched.size,
            )
        }
        item {
            HandoffTimelineSection(
                events = handoffEvents,
                loading = handoffLoading,
            )
        }
        item { FactorySectionHeader(title = stringResource(R.string.mobile_factory_ui_ready_for_loading), count = approved.size) }
        if (approved.isEmpty()) {
            item(span = { GridItemSpan(maxLineSpan) }) { 
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Empty Queue",
                    body = "No approved transfers are waiting at the bay."
                )
            }
        } else {
            items(approved, key = { it.id }) { transfer ->
                TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
            }
        }
        item { FactorySectionHeader(title = stringResource(R.string.mobile_factory_ui_now_loading), count = loadingState.size) }
        if (loadingState.isEmpty()) {
            item(span = { GridItemSpan(maxLineSpan) }) { 
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Empty Queue",
                    body = "Nothing is actively loading right now."
                )
            }
        } else {
            items(loadingState, key = { it.id }) { transfer ->
                TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
            }
        }
        item { FactorySectionHeader(title = stringResource(R.string.supplier_portal_dispatch_text_dispatched), count = dispatched.size) }
        if (dispatched.isEmpty()) {
            item(span = { GridItemSpan(maxLineSpan) }) { 
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Empty Queue",
                    body = "No transfers have been dispatched in the current view."
                )
            }
        } else {
            items(dispatched, key = { it.id }) { transfer ->
                TransferCard(transfer, onClick = { onTransferClick(transfer.id) })
            }
        }
    }
}
