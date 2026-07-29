package com.pegasusx.driver.ui.screens.offload

import androidx.activity.compose.BackHandler
import androidx.activity.compose.LocalOnBackPressedDispatcherOwner
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.HourglassTop
import androidx.compose.material.icons.filled.Payments
import androidx.compose.material.icons.filled.Warning
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.ui.components.DriverGpsBanner
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.ui.theme.formattedAmount

@Composable
fun CashCollectionScreen(
    onComplete: () -> Unit,
    onSplitPayment: ((orderId: String, amount: Long) -> Unit)? = null,
    viewModel: CashCollectionViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current
    var showExitConfirm by remember { mutableStateOf(false) }
    var backConfirmed by remember { mutableStateOf(false) }
    val backDispatcher = LocalOnBackPressedDispatcherOwner.current?.onBackPressedDispatcher

    BackHandler(enabled = state.isCompleting || state.phase == CashFiscalPhase.FISCALIZING) { }
    BackHandler(enabled = !state.isCompleting && state.phase == CashFiscalPhase.COLLECT && !backConfirmed) {
        showExitConfirm = true
    }

    if (showExitConfirm) {
        AlertDialog(
            onDismissRequest = { showExitConfirm = false },
            title = { Text("Leave cash collection?") },
            text = { Text("Cash has not been collected yet. Going back will not complete the delivery.") },
            confirmButton = {
                TextButton(onClick = { showExitConfirm = false }) { Text("Stay") }
            },
            dismissButton = {
                TextButton(onClick = {
                    showExitConfirm = false
                    backConfirmed = true
                    backDispatcher?.onBackPressed()
                }) { Text("Leave") }
            }
        )
    }

    if (state.completed || state.phase == CashFiscalPhase.DONE) {
        onComplete()
        return
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        when (state.phase) {
            CashFiscalPhase.FISCALIZING -> {
                com.pegasusx.driver.ui.screens.offload.components.FiscalizingView(amount = state.amount)
            }
            CashFiscalPhase.FISCAL_FAILED -> {
                com.pegasusx.driver.ui.screens.offload.components.FiscalFailedView(
                    error = state.error,
                    isCompleting = state.isCompleting,
                    onRetryFiscal = { viewModel.retryFiscal() }
                )
            }
            else -> {
                com.pegasusx.driver.ui.screens.offload.components.CollectCashView(
                    amount = state.amount,
                    amountReceivedMinor = state.amountReceivedMinor,
                    amountReceivedInput = state.amountReceivedInput,
                    onAmountReceivedChanged = { viewModel.onAmountReceivedChanged(it) },
                    shortfallMinor = state.shortfallMinor,
                    overageMinor = state.overageMinor,
                    error = state.error,
                    isCompleting = state.isCompleting,
                    cashReceived = state.cashReceived,
                    onRecordSplitPayment = { viewModel.recordSplitPayment() },
                    onCollectCash = { viewModel.collectCash() },
                    onAcknowledgeCashReceived = { viewModel.acknowledgeCashReceived() }
                )
            }
        }
    }

    if (state.showConfirmDialog) {
        AlertDialog(
            onDismissRequest = { viewModel.dismissConfirmDialog() },
            title = { Text("Confirm cash collection?") },
            text = {
                val varianceNote = when {
                    state.shortfallMinor > 0 -> " Shortfall ${state.shortfallMinor.formattedAmount()} will be recorded."
                    state.overageMinor > 0 -> " Overage ${state.overageMinor.formattedAmount()} will be recorded."
                    else -> ""
                }
                Text(
                    "You received ${state.amountReceivedMinor.formattedAmount()} from the retailer " +
                        "(expected ${state.amount.formattedAmount()}).$varianceNote " +
                        "Payment will be captured and a fiscal receipt requested for the received amount."
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.dismissConfirmDialog()
                        viewModel.collectCash()
                    },
                    enabled = !state.isCompleting,
                ) { Text("Capture & Fiscalize") }
            },
            dismissButton = {
                TextButton(onClick = { viewModel.dismissConfirmDialog() }) { Text("Go Back") }
            },
        )
    }
}
