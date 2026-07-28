package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.horizontalScroll
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
import com.pegasusx.supplier.data.model.SUPPLIER_BROADCAST_TEMPLATES
import com.pegasusx.supplier.data.model.SupplierBroadcastRequest
import com.pegasusx.supplier.data.model.SupplierEmpathyAdoption
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierMetricTile
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch

private val broadcastRoles = listOf("ALL", "DRIVER", "RETAILER", "PAYLOAD")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperationsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
    onOpenReplenishmentPolicies: () -> Unit = {},
) {
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
    var templateDate by remember { mutableStateOf("") }
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
                val trimmedTitle = title.trim()
                val trimmedBody = body.trim()
                val scopeId = TokenHolder.supplierId.orEmpty().ifBlank { "supplier" }
                val key = SupplierIdempotencyKeys.broadcast(scopeId, broadcastRole, trimmedTitle, trimmedBody)
                val resp = ops.postBroadcast(
                    SupplierBroadcastRequest(trimmedTitle, trimmedBody, broadcastRole),
                    key,
                )
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
                val reason = bypassReason.trim()
                val key = SupplierIdempotencyKeys.paymentBypass(trimmed, reason)
                val resp = ops.issuePaymentBypass(PaymentBypassRequest(trimmed, reason), key)
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
            PegasusLoadingState("Loading operations…", "Empathy adoption and operator tools")
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

            OperatorBroadcast(
                title = title,
                body = body,
                broadcastRole = broadcastRole,
                templateDate = templateDate,
                broadcasting = broadcasting,
                onTitleChange = { title = it },
                onBodyChange = { body = it },
                onBroadcastRoleChange = { broadcastRole = it },
                onTemplateDateChange = { templateDate = it },
                onBroadcast = { sendBroadcast() },
            )

            ReplenishmentAction(
                replenishing = replenishing,
                onOpenReplenishmentPolicies = onOpenReplenishmentPolicies,
                onTriggerReplenishment = { triggerReplenishment() },
            )

            PaymentBypass(
                orderId = orderId,
                bypassReason = bypassReason,
                bypassToken = bypassToken,
                bypassing = bypassing,
                onOrderIdChange = { orderId = it },
                onBypassReasonChange = { bypassReason = it },
                onShowConfirmChange = { showBypassConfirm = it },
            )
        }
    }
}
