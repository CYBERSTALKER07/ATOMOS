package com.pegasusx.warehouse.ui.screens.operations

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
import com.pegasusx.warehouse.ui.components.WarehouseLoadingState
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
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        if (loading && templates.isEmpty()) {
            WarehouseLoadingState(
                title = "Loading operations",
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

            WarehouseSectionTitle("Broadcast templates")
            Text(
                "Built-in depot starters plus your saved custom messages.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState()),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
            ) {
                templates.forEach { template ->
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        FilterChip(
                            selected = false,
                            onClick = {
                                val (t, b, r) = applyTemplate(template, templateDate, customReason)
                                title = t
                                body = b
                                if (broadcastRoles.contains(r)) broadcastRole = r
                            },
                            label = {
                                val suffix = if (template.source == "custom") " · saved" else ""
                                Text(template.title + suffix, maxLines = 1)
                            },
                        )
                        if (template.source == "custom") {
                            IconButton(onClick = { deleteTemplate(template) }) {
                                Icon(Icons.Default.Close, contentDescription = "Delete ${template.title}")
                            }
                        }
                    }
                }
            }

            HorizontalDivider()
            WarehouseSectionTitle("Send depot broadcast")
            OutlinedTextField(
                value = templateDate,
                onValueChange = { templateDate = it },
                label = { Text("Effective date (optional)") },
                placeholder = { Text("2026-07-01") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = customReason,
                onValueChange = { customReason = it },
                label = { Text("Custom reason (optional)") },
                placeholder = { Text("Bay 2 closed") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text("Title") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            ExposedDropdownMenuBox(expanded = roleExpanded, onExpandedChange = { roleExpanded = it }) {
                OutlinedTextField(
                    value = broadcastRole,
                    onValueChange = {},
                    readOnly = true,
                    label = { Text("Target role") },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = roleExpanded) },
                    modifier = Modifier
                        .menuAnchor()
                        .fillMaxWidth(),
                )
                ExposedDropdownMenu(expanded = roleExpanded, onDismissRequest = { roleExpanded = false }) {
                    broadcastRoles.forEach { role ->
                        androidx.compose.material3.DropdownMenuItem(
                            text = { Text(role) },
                            onClick = {
                                broadcastRole = role
                                roleExpanded = false
                            },
                        )
                    }
                }
            }
            OutlinedTextField(
                value = body,
                onValueChange = { body = it },
                label = { Text("Message") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 4,
            )
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = saveAsTemplate, onCheckedChange = { saveAsTemplate = it })
                Text("Save as custom template for this depot")
            }
            Button(
                onClick = { sendBroadcast() },
                enabled = !broadcasting && !savingTemplate,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (broadcasting || savingTemplate) "Sending…" else "Send broadcast")
            }

            HorizontalDivider()
            WarehouseSectionTitle("Pricing impact preview (read-only)")
            Text(
                "Preview how a proposed retailer price would compare to catalog list price. Does not create overrides.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedTextField(
                value = productId,
                onValueChange = { productId = it },
                label = { Text("Product / SKU ID") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = retailerId,
                onValueChange = { retailerId = it },
                label = { Text("Retailer ID (optional)") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            OutlinedTextField(
                value = proposedPrice,
                onValueChange = { proposedPrice = it.filter { ch -> ch.isDigit() } },
                label = { Text("Proposed price (minor units)") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            )
            if (previewLoading) {
                Text("Loading preview…", style = MaterialTheme.typography.bodySmall)
            }
            preview?.let { p ->
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = PegasusSpacing.sm),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Text("Retailers on SKU: ${p.retailersOnSkuCount}")
                    Text("Active overrides: ${p.activeOverrideCount}")
                    Text("Catalog list price: ${p.catalogListPrice}")
                    Text("Margin delta / unit: ${p.marginDeltaPerUnit}")
                    Text(
                        p.marginEstimateLabel,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (p.readOnly == true) {
                        Text(
                            "Read-only — contact supplier to apply overrides.",
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            }
        }
    }
}
