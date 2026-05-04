package com.pegasus.driver.ui.screens.offload

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Timer
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
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
import com.pegasus.driver.ui.theme.LocalPegasusColors
import com.pegasus.driver.ui.theme.StatusGreen
import com.pegasus.driver.ui.theme.StatusOrange
import com.pegasus.driver.ui.theme.Warning
import kotlinx.coroutines.delay

@Composable
fun ShopClosedWaitingScreen(
    orderId: String,
    onClose: () -> Unit,
    onBypassComplete: () -> Unit,
    onReturnToDepot: () -> Unit,
    viewModel: ShopClosedWaitingViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current

    // Countdown timer (3 minutes escalation window)
    var remainingSeconds by remember { mutableIntStateOf(180) }
    LaunchedEffect(Unit) {
        viewModel.reportShopClosed(orderId)
        while (remainingSeconds > 0) {
            delay(1000L)
            remainingSeconds--
        }
    }

    // React to bypass completion
    LaunchedEffect(state.bypassConfirmed) {
        if (state.bypassConfirmed) onBypassComplete()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
    ) {
        // Header
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 56.dp, start = 8.dp, end = 16.dp, bottom = 8.dp)
        ) {
            IconButton(onClick = onClose) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = lab.fg)
            }
            Text(
                text = "SHOP CLOSED",
                style = MaterialTheme.typography.titleMedium.copy(
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold
                ),
                color = lab.fg
            )
        }

        // Status card
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp)
        ) {
            Icon(
                Icons.Filled.Timer,
                contentDescription = null,
                modifier = Modifier.size(48.dp),
                tint = Warning
            )
            Spacer(Modifier.height(16.dp))

            Text(
                text = when {
                    state.retailerResponse != null -> "Retailer responded: ${state.retailerResponse}"
                    state.escalated -> "Escalated to admin"
                    else -> "Waiting for retailer response..."
                },
                style = MaterialTheme.typography.titleLarge,
                color = lab.fg,
                textAlign = TextAlign.Center
            )
            Spacer(Modifier.height(8.dp))

            // Timer
            if (state.retailerResponse == null && !state.escalated) {
                val minutes = remainingSeconds / 60
                val seconds = remainingSeconds % 60
                Text(
                    text = "Escalation in %d:%02d".format(minutes, seconds),
                    style = MaterialTheme.typography.headlineMedium.copy(
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Bold
                    ),
                    color = if (remainingSeconds < 30) StatusOrange else lab.fg
                )
            }
        }

        Spacer(Modifier.height(16.dp))

        // Retailer response display
        if (state.retailerResponse != null) {
            Text(
                text = when (state.retailerResponse) {
                    "OPEN_NOW" -> "Retailer says they're open now! Proceeding..."
                    "5_MIN" -> "Retailer needs 5 minutes. Please wait."
                    "CALL_ME" -> "Retailer requests a phone call."
                    "CLOSED_TODAY" -> "Shop closed for the day. Escalating to admin..."
                    else -> state.retailerResponse ?: ""
                },
                style = MaterialTheme.typography.bodyLarge,
                color = lab.fg,
                modifier = Modifier.padding(horizontal = 24.dp),
                textAlign = TextAlign.Center
            )
            Spacer(Modifier.height(16.dp))
        }

        // Bypass token section (shown when admin issues bypass)
        if (state.bypassToken != null || state.showBypassInput) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 24.dp)
            ) {
                if (state.bypassToken != null) {
                    Text(
                        text = "Admin issued bypass token:",
                        style = MaterialTheme.typography.bodyMedium,
                        color = lab.fg
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = state.bypassToken ?: "",
                        style = MaterialTheme.typography.headlineLarge.copy(
                            fontFamily = FontFamily.Monospace,
                            fontWeight = FontWeight.Bold,
                            letterSpacing = 8.sp
                        ),
                        color = StatusGreen
                    )
                    Spacer(Modifier.height(16.dp))
                }

                // 6-digit bypass input
                var bypassInput by remember { mutableStateOf("") }
                Text(
                    text = "Enter bypass token:",
                    style = MaterialTheme.typography.labelLarge,
                    color = lab.fg
                )
                Spacer(Modifier.height(8.dp))
                BasicTextField(
                    value = bypassInput,
                    onValueChange = { if (it.length <= 6 && it.all { c -> c.isDigit() }) bypassInput = it },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    textStyle = MaterialTheme.typography.headlineMedium.copy(
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.Bold,
                        color = lab.fg,
                        textAlign = TextAlign.Center,
                        letterSpacing = 6.sp
                    ),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 48.dp)
                )
                Spacer(Modifier.height(16.dp))

                Button(
                    onClick = { viewModel.submitBypass(orderId, bypassInput) },
                    enabled = bypassInput.length == 6 && !state.isSubmitting,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(52.dp),
                    shape = MaterialTheme.shapes.medium,
                    colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
                ) {
                    if (state.isSubmitting) {
                        CircularProgressIndicator(
                            color = MaterialTheme.colorScheme.onPrimary,
                            modifier = Modifier.size(20.dp),
                            strokeWidth = 2.dp
                        )
                    } else {
                        Text("Confirm Bypass Offload", fontWeight = FontWeight.Bold)
                    }
                }
            }
        }

        Spacer(Modifier.weight(1f))

        // Error
        state.error?.let {
            Text(
                text = it,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
            )
        }
    }
}
