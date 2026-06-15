package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardCapitalization
import com.pegasusx.supplier.data.model.PaymentBypassRequest
import com.pegasusx.supplier.data.model.SupplierBroadcastRequest
import com.pegasusx.supplier.data.model.SupplierEmpathyAdoption
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierMetricTile
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

private val broadcastRoles = listOf("ALL", "DRIVER", "RETAILER", "PAYLOAD")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var empathy by remember { mutableStateOf<SupplierEmpathyAdoption?>(null) }
    var title by remember { mutableStateOf("") }
    var body by remember { mutableStateOf("") }
    var broadcastRole by remember { mutableStateOf("ALL") }
    var orderId by remember { mutableStateOf("") }
    var bypassReason by remember { mutableStateOf("") }
    var bypassToken by remember { mutableStateOf<String?>(null) }
    var roleExpanded by remember { mutableStateOf(false) }
    var showBypassConfirm by remember { mutableStateOf(false) }
    var broadcasting by remember { mutableStateOf(false) }
    var replenishing by remember { mutableStateOf(false) }
    var bypassing by remember { mutableStateOf(false) }
    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun loadEmpathy() {
        scope.launch {
            loading = true
            try {
                val resp = ops.getEmpathyAdoption()
                empathy = if (resp.isSuccessful) resp.body() else null
            } catch (_: Exception) {
                empathy = null
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { loadEmpathy() }

    fun sendBroadcast() {
        if (title.isBlank() || body.isBlank()) {
            scope.launch { snackbar.showSnackbar("Title and body are required") }
            return
        }
        broadcasting = true
        scope.launch {
            try {
                val resp = ops.postBroadcast(SupplierBroadcastRequest(title.trim(), body.trim(), broadcastRole))
                if (resp.isSuccessful) {
                    snackbar.showSnackbar("Broadcast sent")
                    title = ""
                    body = ""
                } else {
                    snackbar.showSnackbar("Broadcast failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                broadcasting = false
            }
        }
    }

    fun triggerReplenishment() {
        replenishing = true
        scope.launch {
            try {
                val resp = ops.triggerReplenishment()
                if (resp.isSuccessful) {
                    val status = resp.body()?.status ?: "queued"
                    snackbar.showSnackbar("Replenishment $status")
                } else {
                    snackbar.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                replenishing = false
            }
        }
    }

    fun issueBypass() {
        val trimmed = orderId.trim()
        if (trimmed.isEmpty()) {
            scope.launch { snackbar.showSnackbar("Order ID is required") }
            return
        }
        bypassing = true
        bypassToken = null
        scope.launch {
            try {
                val resp = ops.issuePaymentBypass(PaymentBypassRequest(trimmed, bypassReason.trim()))
                if (resp.isSuccessful) {
                    bypassToken = resp.body()?.bypassToken
                    snackbar.showSnackbar("Bypass token issued")
                } else {
                    snackbar.showSnackbar("Bypass failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                bypassing = false
                showBypassConfirm = false
            }
        }
    }

    if (showBypassConfirm) {
        AlertDialog(
            onDismissRequest = { if (!bypassing) showBypassConfirm = false },
            title = { Text("Issue payment bypass?") },
            text = { Text("Order $orderId must be AWAITING_PAYMENT. Driver receives a one-time bypass token.") },
            confirmButton = {
                TextButton(onClick = { issueBypass() }, enabled = !bypassing) { Text("Issue") }
            },
            dismissButton = {
                TextButton(onClick = { showBypassConfirm = false }, enabled = !bypassing) { Text("Cancel") }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Operations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        if (loading && empathy == null) {
            SupplierLoadingState("Loading operations…", "Empathy adoption and operator tools")
            return@Scaffold
        }
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            empathy?.let { adoption ->
                SupplierSectionTitle("Empathy adoption")
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    SupplierMetricTile("Total", adoption.totalPredictions.toString(), Modifier.weight(1f))
                    SupplierMetricTile("Waiting", adoption.predictionsWaiting.toString(), Modifier.weight(1f))
                    SupplierMetricTile("Fired", adoption.predictionsFired.toString(), Modifier.weight(1f))
                }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    SupplierMetricTile("Dormant", adoption.predictionsDormant.toString(), Modifier.weight(1f))
                    SupplierMetricTile("Rejected", adoption.predictionsRejected.toString(), Modifier.weight(1f))
                }
            }

            SupplierSectionTitle("Operator broadcast")
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text("Title") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = body,
                onValueChange = { body = it },
                label = { Text("Message") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 3,
            )
            ExposedDropdownMenuBox(expanded = roleExpanded, onExpandedChange = { roleExpanded = it }) {
                OutlinedTextField(
                    value = broadcastRole,
                    onValueChange = {},
                    readOnly = true,
                    label = { Text("Target role") },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = roleExpanded) },
                    modifier = Modifier.menuAnchor().fillMaxWidth(),
                )
                ExposedDropdownMenu(expanded = roleExpanded, onDismissRequest = { roleExpanded = false }) {
                    broadcastRoles.forEach { role ->
                        DropdownMenuItem(
                            text = { Text(role) },
                            onClick = {
                                broadcastRole = role
                                roleExpanded = false
                            },
                        )
                    }
                }
            }
            Button(onClick = { sendBroadcast() }, enabled = !broadcasting, modifier = Modifier.fillMaxWidth()) {
                Text(if (broadcasting) "Sending…" else "Send broadcast")
            }

            HorizontalDivider()
            SupplierSectionTitle("Replenishment")
            Text(
                "Opens a warehouse supply request against your primary active warehouse.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = { triggerReplenishment() }, enabled = !replenishing, modifier = Modifier.fillMaxWidth()) {
                Text(if (replenishing) "Triggering…" else "Trigger replenishment")
            }

            HorizontalDivider()
            SupplierSectionTitle("Payment bypass")
            OutlinedTextField(
                value = orderId,
                onValueChange = { orderId = it },
                label = { Text("Order ID (AWAITING_PAYMENT)") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.None),
            )
            OutlinedTextField(
                value = bypassReason,
                onValueChange = { bypassReason = it },
                label = { Text("Reason (optional)") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedButton(
                onClick = { showBypassConfirm = true },
                enabled = !bypassing && orderId.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (bypassing) "Issuing…" else "Issue bypass token")
            }
            bypassToken?.let { token ->
                Text("Driver token: $token", style = MaterialTheme.typography.bodyMedium)
            }
        }
    }
}
