package com.pegasusx.driver.ui.screens.supply

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.data.model.SupplyTransferRow
import com.pegasusx.driver.ui.components.DriverLoadingState
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.StatusPill
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusOrange
import com.pegasusx.driver.ui.theme.StatusRed

private val arriveableStates = setOf("IN_TRANSIT", "IN_TRANSIT_TO_WAREHOUSE", "DISPATCHED")

@Composable
fun SupplyTransfersScreen(
    onBack: () -> Unit,
    viewModel: SupplyTransfersViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 56.dp, start = 8.dp, end = 16.dp, bottom = 8.dp),
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = lab.fg)
            }
            Column {
                Text(
                    text = "FACTORY SUPPLY",
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary,
                    letterSpacing = 1.2.sp,
                )
                Text(
                    text = "Warehouse deliveries",
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg,
                )
            }
        }

        state.error?.let { error ->
            Text(
                text = error,
                color = StatusRed,
                fontSize = 12.sp,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }
        state.successMessage?.let { message ->
            Text(
                text = message,
                color = StatusGreen,
                fontSize = 12.sp,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
            )
        }

        when {
            state.isLoading && state.transfers.isEmpty() -> {
                DriverLoadingState(
                    title = "Loading transfers",
                    body = "Fetching assigned supply legs…",
                    modifier = Modifier.fillMaxSize(),
                )
            }
            state.transfers.isEmpty() -> {
                DriverStatePane(
                    kind = DriverStateKind.Empty,
                    headline = "No supply transfers",
                    body = "Assigned factory→warehouse legs will appear here.",
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(24.dp),
                )
            }
            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(horizontal = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    item {
                        Text(
                            text = "${state.activeCount} active · ${state.transfers.size} total",
                            style = MaterialTheme.typography.labelMedium,
                            color = lab.fgTertiary,
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                    }
                    items(state.transfers, key = { it.transferId }) { transfer ->
                        SupplyTransferCard(
                            transfer = transfer,
                            isArriving = state.isArriving == transfer.transferId,
                            onArrive = { viewModel.markArrived(transfer.transferId) },
                        )
                    }
                    item { Spacer(modifier = Modifier.height(88.dp)) }
                }
            }
        }
    }
}

@Composable
private fun SupplyTransferCard(
    transfer: SupplyTransferRow,
    isArriving: Boolean,
    onArrive: () -> Unit,
) {
    val lab = LocalPegasusColors.current
    val state = transfer.state.uppercase()
    val canArrive = state in arriveableStates

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s16)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = transfer.transferId.takeLast(8).uppercase(),
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg,
                )
                StatusPill(
                    label = state.replace('_', ' '),
                    color = when {
                        state == "ARRIVED" -> StatusGreen
                        canArrive -> StatusOrange
                        else -> lab.fgTertiary
                    },
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "Warehouse ${transfer.warehouseId.takeLast(6)}",
                style = MaterialTheme.typography.bodyMedium,
                color = lab.fgSecondary,
            )
            if (!transfer.supplyRequestId.isNullOrBlank()) {
                Text(
                    text = "Supply ${transfer.supplyRequestId!!.takeLast(8)}",
                    style = MaterialTheme.typography.bodySmall,
                    color = lab.fgTertiary,
                )
            }
            Text(
                text = "Volume ${"%.1f".format(transfer.totalVolumeVu)} VU",
                style = MaterialTheme.typography.bodySmall,
                color = lab.fgTertiary,
            )
            if (canArrive) {
                Spacer(modifier = Modifier.height(12.dp))
                Button(
                    onClick = onArrive,
                    enabled = !isArriving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (isArriving) {
                        CircularProgressIndicator(
                            modifier = Modifier.height(18.dp),
                            strokeWidth = 2.dp,
                        )
                    } else {
                        Text("Mark arrived at warehouse")
                    }
                }
            }
        }
    }
}
