package com.pegasusx.factory.ui.screens.staff

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.CreateStaffRequest
import com.pegasusx.factory.data.model.StaffMember
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.screens.staff.components.StaffList
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.factory.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StaffScreen(
    api: FactoryApi,
    onStaffClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var staff by remember { mutableStateOf<List<StaffMember>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getStaff()
                if (resp.isSuccessful && resp.body() != null) {
                    staff = resp.body()!!.staff
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.SupplyRequestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    val onShift = staff.count { it.status.equals("ON_SHIFT", ignoreCase = true) || it.status.equals("ACTIVE", ignoreCase = true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Staff")
                        Text(
                            text = stringResource(R.string.mobile_factory_ui_operator_roster_and_shift_status),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                    IconButton(onClick = { showCreate = true }) { Icon(Icons.Default.Add, "Add") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading && staff.isEmpty() -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_staff),
                body = "Fetching the current factory operator roster.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null && staff.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load staff",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            staff.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No staff on record",
                body = "There are no staff members registered for this factory yet.",
                actionLabel = "Add staff",
                onAction = { showCreate = true },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> StaffList(
                staff = staff,
                onStaffClick = onStaffClick,
                onShift = onShift,
                innerPadding = innerPadding
            )
        }
    }

    if (showCreate) {
        CreateStaffDialog(
            api = api,
            onDismiss = { showCreate = false },
            onCreated = {
                showCreate = false
                load()
            },
        )
    }
}

@Composable
private fun CreateStaffDialog(
    api: FactoryApi,
    onDismiss: () -> Unit,
    onCreated: () -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("FACTORY_OPERATOR") }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add Staff") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Name") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = role,
                    onValueChange = { role = it },
                    label = { Text("Role") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                if (error != null) {
                    Text(error!!, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    submitting = true
                    error = null
                    scope.launch {
                        try {
                            val resp = api.createStaff(
                                CreateStaffRequest(name = name.trim(), role = role.trim().ifBlank { "FACTORY_OPERATOR" }),
                            )
                            if (resp.isSuccessful) onCreated()
                            else error = "Failed (${resp.code()})"
                        } catch (e: Exception) {
                            error = e.message ?: "Error"
                        } finally {
                            submitting = false
                        }
                    }
                },
                enabled = !submitting && name.isNotBlank(),
            ) {
                if (submitting) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                else Text("Create")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
