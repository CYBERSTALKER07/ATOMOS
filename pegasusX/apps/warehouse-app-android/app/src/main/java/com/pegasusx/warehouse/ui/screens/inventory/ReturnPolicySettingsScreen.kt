package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import com.pegasusx.warehouse.data.model.WarehouseReturnPolicy
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnPolicySettingsScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saved by remember { mutableStateOf(false) }
    var reverseSla by remember { mutableStateOf("24") }
    var canOverride by remember { mutableStateOf(false) }
    var retailerWindow by remember { mutableStateOf("") }
    var supplierId by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun warehouseQueryId(): String? =
        TokenHolder.warehouseId?.takeIf { it.isNotBlank() }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getReturnPolicy(warehouseId = warehouseQueryId())
                if (resp.isSuccessful && resp.body() != null) {
                    val p = resp.body()!!
                    supplierId = p.supplierId
                    reverseSla = p.reverseDockSlaHours?.takeIf { it > 0 }?.toString() ?: "24"
                    canOverride = p.canOverrideRetailerWindow
                    retailerWindow = p.retailerFileWindowHours?.takeIf { it > 0 }?.toString().orEmpty()
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun save() {
        scope.launch {
            saving = true
            error = null
            saved = false
            try {
                val body = WarehouseReturnPolicy(
                    supplierId = supplierId,
                    canOverrideRetailerWindow = canOverride,
                    reverseDockSlaHours = reverseSla.toLongOrNull()?.takeIf { it > 0 },
                    retailerFileWindowHours = if (canOverride) {
                        val hours = retailerWindow.toLongOrNull()
                        if (hours == null || hours < 1) {
                            error = "Retailer file window hours required when override is enabled"
                            saving = false
                            return@launch
                        }
                        hours
                    } else {
                        null
                    },
                )
                val resp = api.putReturnPolicy(
                    body = body,
                    idempotencyKey = WarehouseIdempotencyKeys.returnPolicyPut(supplierId),
                    warehouseId = warehouseQueryId(),
                )
                if (resp.isSuccessful) {
                    saved = true
                } else {
                    error = "Save failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Returns & reverse SLA") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                androidx.compose.material3.CircularProgressIndicator()
            }

            error != null && reverseSla.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .verticalScroll(rememberScrollState())
                    .fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(
                    "Reverse-dock SLA and optional retailer claim-window override. Override may only lengthen the supplier base window.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (error != null) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                }
                if (saved) {
                    Text("Return policy saved.", color = MaterialTheme.colorScheme.primary)
                }

                OutlinedTextField(
                    value = reverseSla,
                    onValueChange = { reverseSla = it.filter { ch -> ch.isDigit() } },
                    label = { Text("Reverse dock SLA (hours)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text(
                        "Override retailer claim filing window (lengthen only)",
                        modifier = Modifier.weight(1f),
                    )
                    Switch(checked = canOverride, onCheckedChange = { canOverride = it })
                }

                if (canOverride) {
                    OutlinedTextField(
                        value = retailerWindow,
                        onValueChange = { retailerWindow = it.filter { ch -> ch.isDigit() } },
                        label = { Text("Retailer file window (hours)") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                }

                Button(
                    onClick = { save() },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (saving) "Saving…" else "Save return policy")
                }
            }
        }
    }
}
