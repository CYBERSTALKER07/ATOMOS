package com.pegasusx.warehouse.ui.screens.cashier

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
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
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CashierBlindCountScreen(
    viewModel: CashierViewModel,
    onBack: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val scrollState = rememberScrollState()
    val haptic = LocalHapticFeedback.current

    var showRRNSheet by remember { mutableStateOf(false) }
    var showCITSheet by remember { mutableStateOf(false) }
    var showAMLSheet by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        viewModel.loadSelectors()
    }

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
            text = "Role 7 Depot Cashier · Dual-Blind Denomination Counting Ritual (so'm)",
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
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Session Information", fontWeight = FontWeight.Bold)
                    TextButton(onClick = { viewModel.toggleManualEntryMode() }) {
                        Text(
                            if (state.manualEntryMode) "Use Live Selectors" else "Manual ID Fallback",
                            style = MaterialTheme.typography.labelMedium,
                        )
                    }
                }

                state.selectorsError?.let { msg ->
                    Text(
                        text = msg,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }

                // Driver Selector or Manual ID Fallback
                if (!state.manualEntryMode && state.driverOptions.isNotEmpty()) {
                    DriverDropdownSelector(
                        options = state.driverOptions,
                        selectedDriverId = state.driverId,
                        loading = state.selectorsLoading,
                        onSelect = { viewModel.selectDriver(it) },
                    )
                } else {
                    OutlinedTextField(
                        value = state.driverId,
                        onValueChange = { viewModel.updateDriverId(it) },
                        label = { Text("Driver ID") },
                        placeholder = { Text("Enter driver ID") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }

                // Route Selector or Manual ID Fallback
                if (!state.manualEntryMode && state.routeOptions.isNotEmpty()) {
                    RouteDropdownSelector(
                        options = state.routeOptions,
                        selectedRouteId = state.routeId,
                        loading = state.selectorsLoading,
                        onSelect = { viewModel.selectRoute(it) },
                    )
                } else {
                    OutlinedTextField(
                        value = state.routeId,
                        onValueChange = { viewModel.updateRouteId(it) },
                        label = { Text("Route / Manifest ID") },
                        placeholder = { Text("Enter route ID") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }

                OutlinedTextField(
                    value = state.bagSealBarcode,
                    onValueChange = { viewModel.updateBagSealBarcode(it) },
                    label = { Text("Bag Seal Barcode") },
                    placeholder = { Text("Scan tamper-evident seal barcode") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedTextField(
                        value = state.depotId,
                        onValueChange = { viewModel.updateDepotId(it) },
                        label = { Text("Depot Node") },
                        placeholder = { Text("From warehouse session") },
                        modifier = Modifier.weight(1f),
                        singleLine = true,
                    )
                    OutlinedTextField(
                        value = state.supplierId,
                        onValueChange = { viewModel.updateSupplierId(it) },
                        label = { Text("Supplier ID") },
                        placeholder = { Text("From auth claims") },
                        modifier = Modifier.weight(1f),
                        singleLine = true,
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
                            text = CashierViewModel.formatWholeSom(exp),
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
                    Text("Banknote Counts (so'm)", fontWeight = FontWeight.Bold)
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
                    val subtotalSom = note * count

                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column {
                            Text(
                                text = CashierViewModel.formatWholeSomMajor(note),
                                fontWeight = FontWeight.SemiBold,
                                fontFamily = FontFamily.Monospace,
                            )
                            Text(
                                text = "Subtotal: ${CashierViewModel.formatWholeSomMajor(subtotalSom)}",
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

        // Counted Total Sticky Summary (strictly whole so'm)
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer),
        ) {
            Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("Total Counted Cash", style = MaterialTheme.typography.labelMedium)
                Text(
                    text = CashierViewModel.formatWholeSom(state.countedCashMinor),
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
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
                defaultSessionId = state.session?.sessionId.orEmpty(),
                onDismiss = { showCITSheet = false },
                onSubmit = { badge, seal, amountMinor, sessionIds ->
                    viewModel.executeCITHandover(badge, seal, amountMinor, sessionIds)
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
                onCheck = { tin, amountMinor ->
                    viewModel.checkAMLExposure(tin, amountMinor)
                },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DriverDropdownSelector(
    options: List<CashierDriverOption>,
    selectedDriverId: String,
    loading: Boolean,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val displayLabel = options.firstOrNull { it.driverId == selectedDriverId }?.label ?: selectedDriverId

    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { if (!loading) expanded = !expanded },
    ) {
        OutlinedTextField(
            value = displayLabel,
            onValueChange = {},
            readOnly = true,
            label = { Text("Driver") },
            placeholder = { Text("Select active driver") },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier
                .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                .fillMaxWidth(),
        )
        ExposedDropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
        ) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.label) },
                    onClick = {
                        expanded = false
                        onSelect(option.driverId)
                    },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RouteDropdownSelector(
    options: List<CashierRouteOption>,
    selectedRouteId: String,
    loading: Boolean,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val displayLabel = options.firstOrNull { it.routeId == selectedRouteId }?.label ?: selectedRouteId

    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { if (!loading) expanded = !expanded },
    ) {
        OutlinedTextField(
            value = displayLabel,
            onValueChange = {},
            readOnly = true,
            label = { Text("Route / Manifest") },
            placeholder = { Text("Select active route") },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier
                .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                .fillMaxWidth(),
        )
        ExposedDropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
        ) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.label) },
                    onClick = {
                        expanded = false
                        onSelect(option.routeId)
                    },
                )
            }
        }
    }
}

@Composable
private fun RRNReclassificationForm(
    onDismiss: () -> Unit,
    onSubmit: (orderId: String, gatewayRrn: String, amountMinor: Long) -> Unit,
) {
    var orderId by remember { mutableStateOf("") }
    var gatewayRrn by remember { mutableStateOf("") }
    var amountSom by remember { mutableStateOf("") }
    var validationError by remember { mutableStateOf<String?>(null) }

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
                placeholder = { Text("Enter order ID") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = gatewayRrn,
                onValueChange = { gatewayRrn = it },
                label = { Text("Gateway RRN (12 digits)") },
                placeholder = { Text("Enter 12-digit gateway RRN") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = amountSom,
                onValueChange = { amountSom = it.filter { ch -> ch.isDigit() } },
                label = { Text("Disputed Amount (so'm)") },
                placeholder = { Text("Enter whole so'm amount") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            validationError?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        val somValue = amountSom.toLongOrNull() ?: 0L
                        if (orderId.isBlank() || gatewayRrn.isBlank() || somValue <= 0L) {
                            validationError = "Order ID, Gateway RRN, and positive so'm amount are required."
                            return@Button
                        }
                        validationError = null
                        onSubmit(orderId.trim(), gatewayRrn.trim(), somValue * 100L)
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
    defaultSessionId: String,
    onDismiss: () -> Unit,
    onSubmit: (badge: String, seal: String, amountMinor: Long, sessionIds: List<String>) -> Unit,
) {
    var courierBadge by remember { mutableStateOf("") }
    var citSeal by remember { mutableStateOf("") }
    var amountSom by remember { mutableStateOf("") }
    var sessionIdsInput by remember(defaultSessionId) { mutableStateOf(defaultSessionId) }
    var validationError by remember { mutableStateOf<String?>(null) }

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
                placeholder = { Text("Enter armored courier badge ID") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = citSeal,
                onValueChange = { citSeal = it },
                label = { Text("CIT Bag Seal Barcode") },
                placeholder = { Text("Scan CIT pouch seal barcode") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = amountSom,
                onValueChange = { amountSom = it.filter { ch -> ch.isDigit() } },
                label = { Text("Handover Amount (so'm)") },
                placeholder = { Text("Enter whole so'm amount") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = sessionIdsInput,
                onValueChange = { sessionIdsInput = it },
                label = { Text("Reconciled Session IDs (Comma-separated)") },
                placeholder = { Text("Enter cashier session IDs") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            validationError?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        val somValue = amountSom.toLongOrNull() ?: 0L
                        val sessions = sessionIdsInput.split(',').map { it.trim() }.filter { it.isNotEmpty() }
                        if (courierBadge.isBlank() || citSeal.isBlank() || somValue <= 0L || sessions.isEmpty()) {
                            validationError = "Badge, seal, session ID(s), and positive so'm amount are required."
                            return@Button
                        }
                        validationError = null
                        onSubmit(courierBadge.trim(), citSeal.trim(), somValue * 100L, sessions)
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
    var tin by remember { mutableStateOf("") }
    var testCashSom by remember { mutableStateOf("") }
    var validationError by remember { mutableStateOf<String?>(null) }

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
                text = if (ceilingMinor > 0) {
                    "Current Exposure: ${CashierViewModel.formatWholeSom(exposureMinor)} / ${CashierViewModel.formatWholeSom(ceilingMinor)}"
                } else {
                    "Current Exposure: ${CashierViewModel.formatWholeSom(exposureMinor)} (Uncapped)"
                },
                style = MaterialTheme.typography.bodySmall,
            )

            OutlinedTextField(
                value = tin,
                onValueChange = { tin = it.filter { ch -> ch.isDigit() } },
                label = { Text("Legal Entity TIN") },
                placeholder = { Text("Enter 9-digit TIN") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = testCashSom,
                onValueChange = { testCashSom = it.filter { ch -> ch.isDigit() } },
                label = { Text("Check Settlement Amount (so'm)") },
                placeholder = { Text("0 (balance check) or whole so'm") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            validationError?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        if (tin.isBlank()) {
                            validationError = "Legal entity TIN is required."
                            return@Button
                        }
                        validationError = null
                        val minor = (testCashSom.toLongOrNull() ?: 0L) * 100L
                        onCheck(tin.trim(), minor)
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
