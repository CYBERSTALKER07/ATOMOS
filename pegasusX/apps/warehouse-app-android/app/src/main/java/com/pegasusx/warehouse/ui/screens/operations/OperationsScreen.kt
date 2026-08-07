package com.pegasusx.warehouse.ui.screens.operations

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.BroadcastTemplate
import com.pegasusx.warehouse.data.model.RetailerOverridePreview
import com.pegasusx.warehouse.data.model.WarehouseBroadcastRequest
import com.pegasusx.warehouse.data.model.WarehouseBroadcastTemplateCreateRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val broadcastRoles = listOf("DRIVER", "RETAILER", "ALL")

private fun applyTemplate(
    template: BroadcastTemplate,
    templateDate: String,
    customReason: String,
): Triple<String, String, String> {
    var body = template.body
    if (body.contains("{date}")) {
        val date = templateDate.trim().ifBlank { "the selected date" }
        body = body.replace("{date}", date)
    }
    if (body.contains("{reason}")) {
        val reason = customReason.trim().ifBlank { "operational delay" }
        body = body.replace("{reason}", reason)
    }
    return Triple(template.title, body, template.defaultRole)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperationsScreen(api: WarehouseApi, onBack: (() -> Unit)? = null) {
    var loading by remember { mutableStateOf(true) }
    var templates by remember { mutableStateOf<List<BroadcastTemplate>>(emptyList()) }
    var title by remember { mutableStateOf("") }
    var body by remember { mutableStateOf("") }
    var broadcastRole by remember { mutableStateOf("DRIVER") }
    var templateDate by remember { mutableStateOf("") }
    var customReason by remember { mutableStateOf("") }
    var saveAsTemplate by remember { mutableStateOf(false) }
    var broadcasting by remember { mutableStateOf(false) }
    var savingTemplate by remember { mutableStateOf(false) }
    var roleExpanded by remember { mutableStateOf(false) }

    var productId by remember { mutableStateOf("") }
    var retailerId by remember { mutableStateOf("") }
    var proposedPrice by remember { mutableStateOf("") }
    var preview by remember { mutableStateOf<RetailerOverridePreview?>(null) }
    var previewLoading by remember { mutableStateOf(false) }

    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun loadTemplates() {
        scope.launch {
            loading = true
            try {
                val resp = api.getBroadcastTemplates()
                templates = if (resp.isSuccessful) resp.body()?.templates.orEmpty() else emptyList()
            } catch (_: Exception) {
                templates = emptyList()
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { loadTemplates() }

    LaunchedEffect(productId, retailerId, proposedPrice) {
        val product = productId.trim()
        val price = proposedPrice.toLongOrNull()
        if (product.isEmpty() || price == null || price <= 0) {
            preview = null
            return@LaunchedEffect
        }
        delay(400)
        previewLoading = true
        try {
            val resp = api.previewRetailerPriceOverride(
                com.pegasusx.warehouse.data.model.RetailerOverridePreviewRequest(
                    retailerId = retailerId.trim().ifBlank { null },
                    productId = product,
                    proposedPrice = price,
                ),
            )
            preview = if (resp.isSuccessful) resp.body() else null
        } catch (_: Exception) {
            preview = null
        } finally {
            previewLoading = false
        }
    }

    fun sendBroadcast() {
        val trimmedTitle = title.trim()
        val trimmedBody = body.trim()
        if (trimmedTitle.isEmpty() || trimmedBody.isEmpty()) {
            scope.launch { snackbar.showSnackbar("Title and message are required") }
            return
        }
        broadcasting = true
        scope.launch {
            try {
                if (saveAsTemplate) {
                    savingTemplate = true
                    val createKey = WarehouseIdempotencyKeys.broadcastTemplateCreate(trimmedTitle, trimmedBody)
                    val createResp = api.createBroadcastTemplate(
                        WarehouseBroadcastTemplateCreateRequest(
                            title = trimmedTitle,
                            body = trimmedBody,
                            defaultRole = broadcastRole,
                            category = "custom",
                        ),
                        createKey,
                    )
                    if (!createResp.isSuccessful) {
                        snackbar.showSnackbar("Failed to save template (${createResp.code()})")
                        return@launch
                    }
                }
                val broadcastKey = WarehouseIdempotencyKeys.broadcast(broadcastRole, trimmedTitle, trimmedBody)
                val resp = api.postBroadcast(
                    WarehouseBroadcastRequest(trimmedTitle, trimmedBody, broadcastRole),
                    broadcastKey,
                )
                if (resp.isSuccessful) {
                    val wh = resp.body()?.warehouseId.orEmpty()
                    snackbar.showSnackbar("Broadcast sent from depot $wh")
                    title = ""
                    body = ""
                    saveAsTemplate = false
                    loadTemplates()
                } else {
                    snackbar.showSnackbar("Broadcast failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                broadcasting = false
                savingTemplate = false
            }
        }
    }

    fun deleteTemplate(template: BroadcastTemplate) {
        if (template.source != "custom") return
        scope.launch {
            try {
                val key = WarehouseIdempotencyKeys.broadcastTemplateDelete(template.id)
                val resp = api.deleteBroadcastTemplate(template.id, key)
                if (resp.isSuccessful) {
                    snackbar.showSnackbar("Custom template removed")
                    loadTemplates()
                } else {
                    snackbar.showSnackbar("Delete failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Delete failed")
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Depot operations") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        if (loading && templates.isEmpty()) {
            PegasusLoadingState(
                title = stringResource(R.string.mobile_warehouse_ui_loading_operations),
                body = "Fetching broadcast templates and depot tools",
                modifier = Modifier.padding(padding),
            )
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
            Text(
                "Depot-scoped broadcasts and read-only pricing impact preview.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            OperationsBroadcastForm(
                templates = templates,
                templateDate = templateDate,
                onTemplateDateChange = { templateDate = it },
                customReason = customReason,
                onCustomReasonChange = { customReason = it },
                title = title,
                onTitleChange = { title = it },
                broadcastRole = broadcastRole,
                onBroadcastRoleChange = { broadcastRole = it },
                body = body,
                onBodyChange = { body = it },
                saveAsTemplate = saveAsTemplate,
                onSaveAsTemplateChange = { saveAsTemplate = it },
                broadcasting = broadcasting,
                savingTemplate = savingTemplate,
                onSelectTemplate = { template ->
                    val (t, b, r) = applyTemplate(template, templateDate, customReason)
                    title = t
                    body = b
                    if (listOf("DRIVER", "RETAILER", "ALL").contains(r)) broadcastRole = r
                },
                onDeleteTemplate = { deleteTemplate(it) },
                onBroadcast = { sendBroadcast() }
            )

            HorizontalDivider()
            
            OperationsPricingPreview(
                productId = productId,
                onProductIdChange = { productId = it },
                retailerId = retailerId,
                onRetailerIdChange = { retailerId = it },
                proposedPrice = proposedPrice,
                onProposedPriceChange = { proposedPrice = it.filter { ch -> ch.isDigit() } },
                previewLoading = previewLoading,
                preview = preview
            )
        }
    }
}
