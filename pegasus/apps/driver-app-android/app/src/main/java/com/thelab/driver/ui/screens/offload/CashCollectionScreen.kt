package com.thelab.driver.ui.screens.offload

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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Payments
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.thelab.driver.ui.theme.LocalLabColors
import com.thelab.driver.ui.theme.StatusGreen
import com.thelab.driver.ui.theme.StatusRed
import com.thelab.driver.ui.theme.formattedAmount

@Composable
fun CashCollectionScreen(
    onComplete: () -> Unit,
    onSplitPayment: ((orderId: String, amount: Long) -> Unit)? = null,
    viewModel: CashCollectionViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalLabColors.current
    var showExitConfirm by remember { mutableStateOf(false) }
    var backConfirmed by remember { mutableStateOf(false) }
    val backDispatcher = LocalOnBackPressedDispatcherOwner.current?.onBackPressedDispatcher

    // Block back press while submitting to prevent duplicate completion
    BackHandler(enabled = state.isCompleting) { /* swallow — submission in flight */ }
    // Show confirmation before allowing exit; once confirmed, handler is disabled so system back proceeds
    BackHandler(enabled = !state.isCompleting && !backConfirmed) { showExitConfirm = true }

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
                    backConfirmed = true         // disables our BackHandler
                    backDispatcher?.onBackPressed() // system back now pops the screen
                }) { Text("Leave") }
            }
        )
    }

    if (state.completed) {
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
        Icon(
            imageVector = Icons.Default.Payments,
            contentDescription = null,
            tint = StatusGreen,
            modifier = Modifier.size(80.dp)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "COLLECT CASH",
            fontSize = 11.sp,
            fontWeight = FontWeight.Black,
            fontFamily = FontFamily.Monospace,
            color = lab.fgTertiary,
            letterSpacing = 1.5.sp
        )

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = state.amount.formattedAmount(),
            fontSize = 38.sp,
            fontWeight = FontWeight.Bold,
            color = lab.fg
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Collect the amount above from the retailer before completing delivery.",
            fontSize = 14.sp,
            color = lab.fgTertiary,
            textAlign = TextAlign.Center
        )

        state.error?.let { error ->
            Spacer(modifier = Modifier.height(12.dp))
            Text(text = error, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
        }

        Spacer(modifier = Modifier.height(48.dp))

        // Edge 35: Split Payment button
        if (onSplitPayment != null) {
            OutlinedButton(
                onClick = { state.orderId?.let { onSplitPayment(it, state.amount) } },
                enabled = !state.isCompleting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
                shape = MaterialTheme.shapes.medium,
            ) {
                Text(
                    text = "Split Payment (Pay Now + Pay Later)",
                    fontWeight = FontWeight.Medium,
                    fontSize = 14.sp
                )
            }
            Spacer(modifier = Modifier.height(12.dp))
        }

        Button(
            onClick = { viewModel.collectCash() },
            enabled = !state.isCompleting,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
            shape = MaterialTheme.shapes.medium,
            colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
        ) {
            if (state.isCompleting) {
                CircularProgressIndicator(
                    color = MaterialTheme.colorScheme.onPrimary,
                    modifier = Modifier.size(20.dp),
                    strokeWidth = 2.dp
                )
            } else {
                Text(
                    text = "Cash Collected — Complete",
                    fontWeight = FontWeight.Bold,
                    fontSize = 15.sp
                )
            }
        }
    }
}
