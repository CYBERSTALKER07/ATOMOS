package com.thelab.retailer.ui.screens.profile

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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import com.thelab.retailer.ui.theme.SoftSquircleShape
import com.thelab.retailer.ui.theme.SquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.outlined.Logout
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.SmartToy
import androidx.compose.material.icons.rounded.Store
import androidx.compose.material.icons.rounded.Sync
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(
    viewModel: ProfileViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    // History/Fresh dialog when enabling global auto-order
    if (uiState.showHistoryDialog) {
        AlertDialog(
            onDismissRequest = viewModel::dismissHistoryDialog,
            title = { Text("Use Previous Analytics?") },
            text = { Text("Use existing order history for predictions, or start fresh? Starting fresh requires at least 2 orders per product.") },
            confirmButton = {
                TextButton(onClick = { viewModel.confirmEnableGlobal(useHistory = true) }) {
                    Text("Use History")
                }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = viewModel::dismissHistoryDialog) {
                        Text("Cancel")
                    }
                    TextButton(onClick = { viewModel.confirmEnableGlobal(useHistory = false) }) {
                        Text("Start Fresh")
                    }
                }
            },
        )
    }

    LazyColumn(
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxSize(),
    ) {
        // ── Profile Header Card ──
        item { ProfileHeaderCard(retailerName = uiState.retailerName, retailerId = uiState.retailerId) }

        // ── Stats Row ──
        item { StatsRow(orderCount = uiState.orderCount, totalSpent = uiState.totalSpent) }

        // ── Empathy Engine ──
        item {
            EmpathyEngineCard(
                globalEnabled = uiState.globalAutoOrderEnabled,
                onGlobalToggle = viewModel::toggleGlobalAutoOrder,
                isUpdating = uiState.isUpdatingSettings,
            )
        }

        // ── Settings Sections ──
        item { SettingsSection() }

        // ── Sign Out ──
        item {
            TextButton(onClick = { /* logout */ }, modifier = Modifier.fillMaxWidth()) {
                Icon(Icons.Outlined.Logout, contentDescription = null, tint = MaterialTheme.colorScheme.error)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Sign Out", color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.labelLarge)
            }
        }

        // ── Version footer ──
        item {
            Text(
                "Pegasus · v1.0.0",
                style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp),
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun ProfileHeaderCard(retailerName: String, retailerId: String) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.fillMaxWidth().padding(24.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Box(
                modifier = Modifier.size(72.dp).clip(CircleShape).background(MaterialTheme.colorScheme.primary),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.Store, contentDescription = null, modifier = Modifier.size(36.dp), tint = MaterialTheme.colorScheme.onPrimary)
            }
            Spacer(modifier = Modifier.height(16.dp))
            Text(retailerName, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(4.dp))
            Text(retailerId, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
        }
    }
}

@Composable
private fun StatsRow(orderCount: Int, totalSpent: Long) {
    val spentDisplay = if (totalSpent >= 1000) "$${String.format("%.1f", totalSpent / 1000.0)}k" else "$$totalSpent"
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(3.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(modifier = Modifier.padding(16.dp)) {
            StatItem(value = "$orderCount", label = "Orders", modifier = Modifier.weight(1f))
            StatItem(value = spentDisplay, label = "Spent", modifier = Modifier.weight(1f))
        }
    }
}

@Composable
private fun StatItem(value: String, label: String, modifier: Modifier = Modifier) {
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(value, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold))
        Spacer(modifier = Modifier.height(2.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
    }
}

@Composable
private fun EmpathyEngineCard(
    globalEnabled: Boolean,
    onGlobalToggle: (Boolean) -> Unit,
    isUpdating: Boolean,
) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier.size(36.dp).clip(SquircleShape)
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Rounded.SmartToy, contentDescription = null, modifier = Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                }
                Spacer(modifier = Modifier.width(10.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text("Empathy Engine", style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold))
                    Text("AI-powered automatic reordering", style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
                }
            }

            Spacer(modifier = Modifier.height(14.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(14.dp))

            Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Rounded.Sync, contentDescription = null, modifier = Modifier.size(16.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
                Spacer(modifier = Modifier.width(8.dp))
                Column(modifier = Modifier.weight(1f)) {
                    Text("Auto-Order", style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium))
                    Text(
                        if (globalEnabled) "Active — orders placed automatically" else "Disabled",
                        style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                    )
                }
                Switch(
                    checked = globalEnabled,
                    onCheckedChange = onGlobalToggle,
                    enabled = !isUpdating,
                    colors = SwitchDefaults.colors(
                        checkedTrackColor = MaterialTheme.colorScheme.primary,
                        checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                    ),
                )
            }
        }
    }
}

@Composable
private fun SettingsSection() {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(3.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column {
            SettingsListItem(icon = Icons.Outlined.Settings, title = "General Settings", subtitle = "Language, preferences", onClick = { })
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Rounded.Person, title = "Account", subtitle = "Manage your business details", onClick = { })
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.Notifications, title = "Notifications", subtitle = "Push, email, SMS", onClick = { })
        }
    }
}

@Composable
private fun SettingsListItem(icon: ImageVector, title: String, subtitle: String, onClick: () -> Unit) {
    Surface(onClick = onClick, color = Color.Transparent) {
        Row(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 14.dp), verticalAlignment = Alignment.CenterVertically) {
            Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f), modifier = Modifier.size(20.dp))
            Spacer(modifier = Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(title, style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium))
                Text(subtitle, style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f))
            }
            Icon(Icons.Outlined.ChevronRight, contentDescription = null, tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f), modifier = Modifier.size(18.dp))
        }
    }
}
