package com.thelab.warehouse.ui.screens.drivers

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.thelab.warehouse.data.model.CreateDriverRequest
import com.thelab.warehouse.data.model.Driver
import com.thelab.warehouse.data.remote.WarehouseApi
import com.thelab.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DriversScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var drivers by remember { mutableStateOf<List<Driver>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var createdPin by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getDrivers()
                if (resp.isSuccessful && resp.body() != null) drivers = resp.body()!!.drivers
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Drivers") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                    IconButton(onClick = { showCreate = true }) { Icon(Icons.Default.Add, "Add") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(drivers, key = { it.driverId }) { driver ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(driver.name, style = MaterialTheme.typography.titleSmall)
                                Text(driver.phone, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            AssistChip(
                                onClick = {},
                                label = { Text(driver.truckStatus.ifBlank { "IDLE" }, style = MaterialTheme.typography.labelSmall) },
                            )
                        }
                    }
                }
            }
        }
    }

    // Create driver dialog
    if (showCreate) {
        CreateDriverDialog(
            api = api,
            onDismiss = { showCreate = false },
            onCreated = { pin ->
                createdPin = pin
                showCreate = false
                load()
            },
        )
    }

    // PIN display dialog
    if (createdPin != null) {
        AlertDialog(
            onDismissRequest = { createdPin = null },
            title = { Text("Driver Created") },
            text = {
                Column {
                    Text("One-time PIN — save it now:")
                    Spacer(Modifier.height(LabSpacing.md))
                    Text(createdPin!!, style = MaterialTheme.typography.headlineMedium, color = MaterialTheme.colorScheme.primary)
                }
            },
            confirmButton = { TextButton(onClick = { createdPin = null }) { Text("Done") } },
        )
    }
}

@Composable
private fun CreateDriverDialog(
    api: WarehouseApi,
    onDismiss: () -> Unit,
    onCreated: (String) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var submitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add Driver") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = phone, onValueChange = { phone = it }, label = { Text("Phone") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                if (error != null) Text(error!!, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    submitting = true; error = null
                    scope.launch {
                        try {
                            val resp = api.createDriver(CreateDriverRequest(name = name, phone = phone))
                            if (resp.isSuccessful && resp.body() != null) onCreated(resp.body()!!.pin)
                            else error = "Failed (${resp.code()})"
                        } catch (e: Exception) { error = e.message ?: "Error" }
                        finally { submitting = false }
                    }
                },
                enabled = !submitting && name.isNotBlank() && phone.isNotBlank(),
            ) {
                if (submitting) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                else Text("Create")
            }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
