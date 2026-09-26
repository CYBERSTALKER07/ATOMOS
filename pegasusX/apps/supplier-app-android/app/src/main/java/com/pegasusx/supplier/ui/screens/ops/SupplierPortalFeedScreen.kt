package com.pegasusx.supplier.ui.screens.ops

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import com.pegasusx.supplier.data.model.CreditAdminDisableRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import retrofit2.Response

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplierPortalFeedScreen(
    title: String,
    onBack: () -> Unit,
    loader: suspend () -> Response<JsonElement>,
) {
    var body by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = loader()
                if (resp.isSuccessful) {
                    body = resp.body()?.toString() ?: "{}"
                } else {
                    error = if (resp.code() == 503) "Unavailable" else "Failed (${resp.code()})"
                    body = null
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
                body = null
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(title) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = title,
                body = "Loading live API.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && body == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(body ?: "{}", modifier = Modifier.fillMaxWidth())
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditAdminDisableScreen(
    api: SupplierApi,
    onBack: () -> Unit,
) {
    var mode by remember { mutableStateOf("relationship") }
    var supplierId by remember { mutableStateOf("") }
    var retailerId by remember { mutableStateOf("") }
    var ticketId by remember { mutableStateOf("") }
    var reason by remember { mutableStateOf("") }
    var msg by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val canSubmit = supplierId.isNotBlank() && ticketId.isNotBlank() && reason.isNotBlank() &&
        (mode != "relationship" || retailerId.isNotBlank())

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Credit admin disable") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(PegasusSpacing.lg)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text("Permanent disable requires ticket_id + reason. 403 unless PLATFORM_ADMIN.")
            OutlinedTextField(
                value = mode,
                onValueChange = { mode = it },
                label = { Text("Mode: relationship | program") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = supplierId,
                onValueChange = { supplierId = it },
                label = { Text("Supplier ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            if (mode != "program") {
                OutlinedTextField(
                    value = retailerId,
                    onValueChange = { retailerId = it },
                    label = { Text("Retailer ID") },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            OutlinedTextField(
                value = ticketId,
                onValueChange = { ticketId = it },
                label = { Text("Ticket ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = reason,
                onValueChange = { reason = it },
                label = { Text("Reason") },
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                enabled = canSubmit && !busy,
                onClick = {
                    busy = true
                    msg = null
                    scope.launch {
                        try {
                            val body = CreditAdminDisableRequest(ticketId = ticketId, reason = reason)
                            val resp = if (mode == "program") {
                                api.adminDisableCreditProgram(supplierId, body)
                            } else {
                                api.adminDisableCreditRelationship(supplierId, retailerId, body)
                            }
                            msg = if (resp.isSuccessful) {
                                resp.body()?.toString() ?: "disabled"
                            } else {
                                "Failed (${resp.code()})"
                            }
                        } catch (e: Exception) {
                            msg = e.message ?: "Network error"
                        } finally {
                            busy = false
                        }
                    }
                },
            ) { Text("Disable permanently") }
            msg?.let { Text(it) }
        }
    }
}
