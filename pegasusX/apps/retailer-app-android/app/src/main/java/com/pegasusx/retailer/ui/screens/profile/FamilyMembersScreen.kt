package com.pegasusx.retailer.ui.screens.profile

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
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Delete
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.SwapHoriz
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.components.PegasusEmptyState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FamilyMembersScreen(
    onNavigateBack: () -> Unit,
    onOpenTeam: (() -> Unit)? = null,
    viewModel: FamilyMembersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var showAddDialog by remember { mutableStateOf(false) }
    var showMigrateConfirm by remember { mutableStateOf(false) }
    val clipboard = LocalClipboardManager.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Family Members") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(imageVector = Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    if (onOpenTeam != null) {
                        TextButton(onClick = onOpenTeam) { Text("Team") }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface,
                ),
            )
        },
        floatingActionButton = {
            if (!uiState.familyGone) {
                FloatingActionButton(onClick = { showAddDialog = true }) {
                    Icon(Icons.Rounded.Add, contentDescription = "Add Member")
                }
            }
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            val syncMessage = when {
                uiState.loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                uiState.banner != null -> uiState.banner
                uiState.isLoading && uiState.members.isNotEmpty() -> "Syncing family/staff members..."
                else -> null
            }

            if (syncMessage != null) {
                val loadIssue = uiState.loadIssue
                val containerColor = when (loadIssue) {
                    FamilyMembersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    FamilyMembersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    FamilyMembersLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    FamilyMembersLoadIssue.GONE -> MaterialTheme.colorScheme.secondaryContainer.copy(alpha = 0.5f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    FamilyMembersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    FamilyMembersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    FamilyMembersLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                    FamilyMembersLoadIssue.GONE -> MaterialTheme.colorScheme.onSecondaryContainer
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
                    if (loadIssue != null && loadIssue != FamilyMembersLoadIssue.GONE) {
                        TextButton(onClick = viewModel::loadData) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }

            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                item(span = { GridItemSpan(maxLineSpan) }) {
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.35f),
                        ),
                    ) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text(
                                "Migrate Family → Team",
                                style = MaterialTheme.typography.titleMedium,
                            )
                            Text(
                                "Converts contacts with a phone into Team RECEIVER accounts. " +
                                    "Temp passwords show once. Family add closes after migrate.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Button(
                                onClick = { showMigrateConfirm = true },
                                enabled = !uiState.migrating && !uiState.familyGone && uiState.members.isNotEmpty(),
                            ) {
                                if (uiState.migrating) {
                                    CircularProgressIndicator(
                                        modifier = Modifier.height(18.dp).padding(end = 8.dp),
                                        strokeWidth = 2.dp,
                                    )
                                } else {
                                    Icon(Icons.Rounded.SwapHoriz, contentDescription = null)
                                    Spacer(Modifier.padding(4.dp))
                                }
                                Text(
                                    when {
                                        uiState.familyGone -> "Already migrated"
                                        uiState.migrating -> "Migrating…"
                                        else -> "Migrate to Team"
                                    },
                                )
                            }
                        }
                    }
                }

                uiState.migrateResult?.let { result ->
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Card {
                            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                Text(
                                    "Migration result: ${result.migrated.size} migrated · " +
                                        "${result.skipped.size} skipped · ${result.familyRemaining} remaining",
                                    style = MaterialTheme.typography.titleSmall,
                                )
                                result.migrated.forEach { m ->
                                    Row(
                                        Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically,
                                    ) {
                                        Column(Modifier.weight(1f)) {
                                            Text(m.name, style = MaterialTheme.typography.bodyMedium)
                                            Text(
                                                "${m.phone} · ${m.retailerRole}",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            )
                                            m.tempPassword?.let { pw ->
                                                Text(pw, style = MaterialTheme.typography.bodySmall)
                                            }
                                        }
                                        m.tempPassword?.let { pw ->
                                            TextButton(onClick = {
                                                clipboard.setText(AnnotatedString(pw))
                                            }) { Text("Copy") }
                                        }
                                    }
                                }
                                result.skipped.forEach { s ->
                                    Text(
                                        "${s.phone ?: s.memberId}: ${s.reason}",
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.error,
                                    )
                                }
                                if (onOpenTeam != null) {
                                    TextButton(onClick = onOpenTeam) { Text("Manage in Team") }
                                }
                            }
                        }
                    }
                }

                if (uiState.isLoading && uiState.members.isEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Box(Modifier.fillMaxWidth().padding(32.dp), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator()
                        }
                    }
                } else if (uiState.members.isEmpty()) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        PegasusEmptyState(
                            icon = Icons.Rounded.Person,
                            title = when (uiState.loadIssue) {
                                FamilyMembersLoadIssue.RESTRICTED -> "Family/Staff Access Restricted"
                                FamilyMembersLoadIssue.OFFLINE -> "Family/Staff Offline"
                                FamilyMembersLoadIssue.ERROR -> "Family/Staff Unavailable"
                                FamilyMembersLoadIssue.GONE -> "Family list empty"
                                null -> "No Staff/Family"
                            },
                            message = uiState.error
                                ?: if (uiState.familyGone) {
                                    "Use Team to manage staff."
                                } else {
                                    "Add members with a phone, then migrate to Team."
                                },
                        )
                    }
                } else {
                    items(uiState.members, key = { it.id }) { member ->
                        Card(
                            modifier = Modifier.fillMaxWidth(),
                            colors = CardDefaults.cardColors(
                                containerColor = MaterialTheme.colorScheme.surfaceContainer,
                            ),
                        ) {
                            Row(
                                modifier = Modifier.fillMaxWidth().padding(16.dp),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column {
                                    Text(member.name, style = MaterialTheme.typography.titleMedium)
                                    Text(
                                        member.phone.ifBlank { "No phone — skipped on migrate" },
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = if (member.phone.isBlank()) {
                                            MaterialTheme.colorScheme.error
                                        } else {
                                            MaterialTheme.colorScheme.onSurfaceVariant
                                        },
                                    )
                                }
                                IconButton(onClick = { viewModel.deleteMember(member.id) }) {
                                    Icon(
                                        Icons.Rounded.Delete,
                                        contentDescription = "Delete",
                                        tint = MaterialTheme.colorScheme.error,
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showMigrateConfirm) {
        AlertDialog(
            onDismissRequest = { showMigrateConfirm = false },
            title = { Text("Migrate to Team?") },
            text = {
                Text(
                    "Contacts with a phone become Team RECEIVER accounts. " +
                        "Temporary passwords appear once — copy them before leaving.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    showMigrateConfirm = false
                    viewModel.migrateToTeam()
                }) { Text("Migrate") }
            },
            dismissButton = {
                TextButton(onClick = { showMigrateConfirm = false }) { Text("Cancel") }
            },
        )
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
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = phone,
                        onValueChange = { phone = it },
                        label = { Text("Phone (required for Team migrate)") },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    if (name.isNotBlank()) {
                        viewModel.addMember(name.trim(), phone.trim())
                        showAddDialog = false
                    }
                }) { Text("Add") }
            },
            dismissButton = {
                TextButton(onClick = { showAddDialog = false }) { Text("Cancel") }
            },
        )
    }
}
