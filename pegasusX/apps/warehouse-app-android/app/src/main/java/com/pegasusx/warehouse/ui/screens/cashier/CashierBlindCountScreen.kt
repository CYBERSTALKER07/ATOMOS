package com.pegasusx.warehouse.ui.screens.cashier

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.size
import java.text.NumberFormat
import java.util.Locale

@Composable
fun CashierBlindCountScreen(
    viewModel: CashierViewModel,
    onBack: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val state by viewModel.uiState.collectAsState()
    val scrollState = rememberScrollState()
    val uzsFormatter = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }
    val haptic = LocalHapticFeedback.current

    var showRRNSheet by remember { mutableStateOf(false) }
    var showCITSheet by remember { mutableStateOf(false) }
    var showAMLSheet by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "Depot Cashier & Vault",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
            )
            OutlinedButton(onClick = onBack) {
                Text("Back")
            }
        }

        Text(
            text = "Role 7 Depot Cashier · Dual-Blind Denomination Counting Ritual",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        // Dual-Blind Recount Ratchet Alert Banner (Attempt #2 required)
        if (state.isRecountRequired) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .background(Color(0xFFE65100))
                    .padding(16.dp),
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Warning,
                            contentDescription = "Recount Alert",
                            tint = Color.White,
                        )
                        Text(
                            text = "MANDATORY BLIND RECOUNT #2 REQUIRED",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            fontSize = 15.sp,
                        )
                    }
                    Text(
                        text = "Physical count discrepancy detected. Expected amount remains strictly confidential. Clear the banknote tray and perform count #2.",
                        color = Color.White,
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }
        }

        // Clearance Badge when Balanced
        if (state.isBalanced) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .background(Color(0xFF1B5E20))
                    .padding(16.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = "Balanced",
                        tint = Color.White,
                    )
                    Column {
                        Text(
                            text = "SESSION BALANCED & CLEARED",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                        )
                        state.clearanceTokenHash?.let { hash ->
                            Text(
                                text = "Clearance Token: ${hash.take(16)}...",
                                color = Color.White,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }

        // Session & Expected Cash Card (Strict Masking)
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("Session Information", fontWeight = FontWeight.Bold)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = state.driverId,
                        onValueChange = { viewModel.updateDriverId(it) },
                        label = { Text("Driver ID") },
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = state.bagSealBarcode,
                        onValueChange = { viewModel.updateBagSealBarcode(it) },
                        label = { Text("Bag Seal Barcode") },
                        modifier = Modifier.weight(1f),
                    )
                }

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Expected Cash:")
                    if (state.isBlindCountMasked) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                imageVector = Icons.Default.Lock,
                                contentDescription = "Masked",
                                tint = MaterialTheme.colorScheme.primary,
                            )
                            Spacer(Modifier.width(4.dp))
                            Text(
                                text = "CONFIDENTIAL (MASKED)",
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }
                    } else {
                        val exp = state.session?.expectedCashMinor ?: 0L
                        Text(
                            text = "${uzsFormatter.format(exp / 100)} UZS (${exp} Tiyin)",
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }

                if (state.session == null) {
                    Button(
                        onClick = { viewModel.startSession() },
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !state.isLoading,
                    ) {
                        Text("Register Deposit Bag")
                    }
                } else {
                    Text(
                        text = "Session: ${state.session?.sessionId} · Attempt #${state.attemptNumber}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        // Banknote Denomination Counting Grid
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Banknote Counts (Uzbek Som)", fontWeight = FontWeight.Bold)
                    OutlinedButton(onClick = {
                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                        viewModel.clearCounts()
                    }) {
                        Text("Clear")
                    }
                }
                HorizontalDivider()

                val notes = listOf(200000L, 100000L, 50000L, 20000L, 10000L, 5000L, 2000L, 1000L)
                notes.forEach { note ->
                    val count = state.denominations[note] ?: 0L
                    val subtotalUzs = note * count
                    val subtotalTiyin = subtotalUzs * 100L

                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column {
                            Text(
                                text = "${uzsFormatter.format(note)} UZS",
                                fontWeight = FontWeight.SemiBold,
                                fontFamily = FontFamily.Monospace,
                            )
                            Text(
                                text = "Subtotal: ${uzsFormatter.format(subtotalUzs)} UZS (${subtotalTiyin} Tiyin)",
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = FontFamily.Monospace,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }

                        Row(verticalAlignment = Alignment.CenterVertically) {
                            OutlinedButton(
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                    viewModel.adjustDenominationCount(note, -1L)
                                },
                                modifier = Modifier.size(48.dp),
                                contentPadding = PaddingValues(0.dp),
                            ) {
                                Text("-", fontSize = 20.sp, fontWeight = FontWeight.Bold)
                            }
                            Text(
                                text = "$count",
                                modifier = Modifier.padding(horizontal = 12.dp),
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace,
                            )
                            OutlinedButton(
                                onClick = {
                                    haptic.performHapticFeedback(HapticFeedbackType.TextHandleMove)
                                    viewModel.adjustDenominationCount(note, 1L)
                                },
                                modifier = Modifier.size(48.dp),
                                contentPadding = PaddingValues(0.dp),
                            ) {
                                Text("+", fontSize = 20.sp, fontWeight = FontWeight.Bold)
                            }
                        }
                    }
                }
            }
        }

        // Counted Total Sticky Summary
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("Total Counted Cash", style = MaterialTheme.typography.labelMedium)
                val countedUzs = state.countedCashMinor / 100L
                Text(
                    text = "${uzsFormatter.format(countedUzs)} UZS",
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                )
                Text(
                    text = "${state.countedCashMinor} Tiyin (Integer Minor Units)",
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    color = MaterialTheme.colorScheme.onPrimaryContainer,
                )
            }
        }

        // Status or Error
        state.statusMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
        }
        state.errorMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold)
        }

        // Submission Actions
        Button(
            onClick = {
                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                viewModel.submitBlindCount()
            },
            modifier = Modifier.fillMaxWidth().height(52.dp),
            enabled = !state.isLoading && state.session != null,
        ) {
            if (state.isLoading) {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp), strokeWidth = 2.dp)
            }
            Text(
                if (state.isRecountRequired) "Submit Recount #2" else "Submit Blind Count",
                fontWeight = FontWeight.Bold,
                fontSize = 16.sp,
            )
        }

        // Auxiliary Operations: RRN Reclassification, CIT Handover, AML Check
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            OutlinedButton(
                onClick = { showRRNSheet = true },
                modifier = Modifier.weight(1f),
            ) {
                Text("RRN Reclass")
            }
            OutlinedButton(
                onClick = { showCITSheet = true },
                modifier = Modifier.weight(1f),
            ) {
                Text("CIT Vault")
            }
            OutlinedButton(
                onClick = { showAMLSheet = true },
                modifier = Modifier.weight(1f),
            ) {
                Text("AML Check")
            }
        }

        // RRN Reclassification Section
        if (showRRNSheet) {
            RRNReclassificationForm(
                onDismiss = { showRRNSheet = false },
                onSubmit = { orderId, rrn, amountMinor ->
                    viewModel.reclassifyTenderRRN(orderId, rrn, amountMinor)
                    showRRNSheet = false
                },
            )
        }

        // CIT Handover Section
        if (showCITSheet) {
            CITHandoverForm(
                onDismiss = { showCITSheet = false },
                onSubmit = { badge, seal, amount ->
                    viewModel.executeCITHandover(badge, seal, amount)
                    showCITSheet = false
                },
            )
        }

        // AML Exposure Check Section
        if (showAMLSheet) {
            AMLCheckForm(
                exposureMinor = state.amlExposureMinor,
                ceilingMinor = state.amlCeilingMinor,
                onDismiss = { showAMLSheet = false },
                onCheck = { tin, amount ->
                    viewModel.checkAMLExposure(tin, amount)
                },
            )
        }
    }
}

@Composable
private fun RRNReclassificationForm(
    onDismiss: () -> Unit,
    onSubmit: (orderId: String, gatewayRrn: String, amountMinor: Long) -> Unit,
) {
    var orderId by remember { mutableStateOf("order-tashkent-701") }
    var gatewayRrn by remember { mutableStateOf("RRN-HUMO-991288") }
    var amountUzs by remember { mutableStateOf("150000") }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("RRN Payment Reclassification", fontWeight = FontWeight.Bold)
            Text(
                text = "Reclassify mistaken Cash ePOD entry to Humo/Uzcard terminal payment without mutating historical logs.",
                style = MaterialTheme.typography.bodySmall,
            )
            OutlinedTextField(
                value = orderId,
                onValueChange = { orderId = it },
                label = { Text("Order ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = gatewayRrn,
                onValueChange = { gatewayRrn = it },
                label = { Text("Gateway RRN (12 digits)") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = amountUzs,
                onValueChange = { amountUzs = it },
                label = { Text("Disputed Amount (UZS)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        val minor = (amountUzs.toLongOrNull() ?: 0L) * 100L
                        onSubmit(orderId, gatewayRrn, minor)
                    },
                    modifier = Modifier.weight(1f),
                ) {
                    Text("Apply Reclassification")
                }
                OutlinedButton(onClick = onDismiss) {
                    Text("Cancel")
                }
            }
        }
    }
}

@Composable
private fun CITHandoverForm(
    onDismiss: () -> Unit,
    onSubmit: (badge: String, seal: String, amountMinor: Long) -> Unit,
) {
    var courierBadge by remember { mutableStateOf("INKAS-TASH-44") }
    var citSeal by remember { mutableStateOf("CIT-SEAL-8812") }
    var amountUzs by remember { mutableStateOf("10000000") }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("CIT Armored Vault Handover", fontWeight = FontWeight.Bold)
            Text("Consolidate balanced sessions for armored Inkassatsiya pickup.", style = MaterialTheme.typography.bodySmall)
            OutlinedTextField(
                value = courierBadge,
                onValueChange = { courierBadge = it },
                label = { Text("Courier Badge ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = citSeal,
                onValueChange = { citSeal = it },
                label = { Text("CIT Bag Seal Barcode") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = amountUzs,
                onValueChange = { amountUzs = it },
                label = { Text("Handover Amount (UZS)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        val minor = (amountUzs.toLongOrNull() ?: 0L) * 100L
                        onSubmit(courierBadge, citSeal, minor)
                    },
                    modifier = Modifier.weight(1f),
                ) {
                    Text("Execute Handover")
                }
                OutlinedButton(onClick = onDismiss) {
                    Text("Cancel")
                }
            }
        }
    }
}

@Composable
private fun AMLCheckForm(
    exposureMinor: Long,
    ceilingMinor: Long,
    onDismiss: () -> Unit,
    onCheck: (tin: String, amountMinor: Long) -> Unit,
) {
    var tin by remember { mutableStateOf("304899122") }
    var testCashUzs by remember { mutableStateOf("5000000") }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
    ) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("AML Daily Cash Exposure (Uncapped Policy)", fontWeight = FontWeight.Bold)
            val progress = if (ceilingMinor > 0) (exposureMinor.toFloat() / ceilingMinor.toFloat()).coerceIn(0f, 1f) else 1f
            LinearProgressIndicator(
                progress = { progress },
                modifier = Modifier.fillMaxWidth(),
            )
            Text(
                text = if (ceilingMinor > 0) "Current Exposure: ${exposureMinor / 100} / ${ceilingMinor / 100} UZS" else "Current Exposure: ${exposureMinor / 100} UZS (Uncapped)",
                style = MaterialTheme.typography.bodySmall,
            )

            OutlinedTextField(
                value = tin,
                onValueChange = { tin = it },
                label = { Text("Legal Entity TIN") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = testCashUzs,
                onValueChange = { testCashUzs = it },
                label = { Text("Check Settlement Amount (UZS)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        val minor = (testCashUzs.toLongOrNull() ?: 0L) * 100L
                        onCheck(tin, minor)
                    },
                    modifier = Modifier.weight(1f),
                ) {
                    Text("Verify AML Limit")
                }
                OutlinedButton(onClick = onDismiss) {
                    Text("Close")
                }
            }
        }
    }
}
