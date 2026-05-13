package com.pegasus.retailer.ui.screens.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.ui.components.PegasusEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FamilyMembersScreen(
    onNavigateBack: () -> Unit,
    viewModel: FamilyMembersViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    var showAddDialog by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Family Members") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(imageVector = Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        },
        floatingActionButton = {
            FloatingActionButton(onClick = { showAddDialog = true }) {
                Icon(Icons.Rounded.Add, contentDescription = "Add Member")
            }
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            val syncMessage = when {
                uiState.loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                uiState.isLoading && uiState.members.isNotEmpty() -> "Syncing family/staff members..."
                else -> null
            }

            if (syncMessage != null) {
                val loadIssue = uiState.loadIssue
                val containerColor = when (loadIssue) {
                    FamilyMembersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    FamilyMembersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    FamilyMembersLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    FamilyMembersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    FamilyMembersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    FamilyMembersLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                    null -> MaterialTheme.colorScheme.onPrimaryContainer
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .clip(MaterialTheme.shapes.medium)
                        .background(containerColor)
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = syncMessage,
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.labelMedium,
                        color = contentColor,
                    )
                    if (loadIssue != null) {
                        TextButton(onClick = viewModel::loadData) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }

            Box(modifier = Modifier.fillMaxSize()) {
                if (uiState.isLoading && uiState.members.isEmpty()) {
                    CircularProgressIndicator(modifier = Modifier.align(Alignment.Center))
                } else if (uiState.members.isEmpty()) {
                    PegasusEmptyState(
                        icon = Icons.Rounded.Person,
                        title = when (uiState.loadIssue) {
                            FamilyMembersLoadIssue.RESTRICTED -> "Family/Staff Access Restricted"
                            FamilyMembersLoadIssue.OFFLINE -> "Family/Staff Offline"
                            FamilyMembersLoadIssue.ERROR -> "Family/Staff Unavailable"
                            null -> "No Staff/Family"
                        },
                        message = uiState.error ?: "Add members to allow them to manage operations",
                        modifier = Modifier.align(Alignment.Center)
                    )
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        items(uiState.members) { member ->
                            Card(
                                modifier = Modifier.fillMaxWidth(),
                                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainer)
                            ) {
                                Row(
                                    modifier = Modifier.fillMaxWidth().padding(16.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Column {
                                        Text(member.name, style = MaterialTheme.typography.titleMedium)
                                        Text(member.phone, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                    }
                                    IconButton(onClick = { viewModel.deleteMember(member.id) }) {
                                        Icon(Icons.Rounded.Delete, contentDescription = "Delete", tint = MaterialTheme.colorScheme.error)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showAddDialog) {
        var name by remember { mutableStateOf("") }
        var phone by remember { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { showAddDialog = false },
            title = { Text("Add Member") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it },
                        label = { Text("Name") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = phone,
                        onValueChange = { phone = it },
                        label = { Text("Phone") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (name.isNotBlank() && phone.isNotBlank()) {
                        viewModel.addMember(name, phone)
                        showAddDialog = false
                    }
                }) {
                    Text("Add")
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddDialog = false }) { Text("Cancel") }
            }
        )
    }
}
