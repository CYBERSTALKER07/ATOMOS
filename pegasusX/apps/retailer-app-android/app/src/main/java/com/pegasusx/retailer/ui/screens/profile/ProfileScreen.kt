package com.pegasusx.retailer.ui.screens.profile

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.retailer.ui.components.RetailerSectionHeader
import com.pegasusx.retailer.ui.theme.PegasusSpacing
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.SquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material.icons.outlined.Logout
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.People
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
import com.pegasusx.retailer.ui.screens.profile.components.ProfileHeaderCard
import com.pegasusx.retailer.ui.screens.profile.components.StatsRow
import com.pegasusx.retailer.ui.screens.profile.components.EmpathyEngineCard
import com.pegasusx.retailer.ui.screens.profile.components.SettingsSection
import com.pegasusx.retailer.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(
    onAccountClick: () -> Unit = {},
    onFamilyMembersClick: () -> Unit = {},
    onCapabilitiesClick: () -> Unit = {},
    onTeamClick: () -> Unit = {},
    onLocationsClick: () -> Unit = {},
    onStoreStockClick: () -> Unit = {},
    onLocalSkusClick: () -> Unit = {},
    onPosClick: () -> Unit = {},
    onShiftsClick: () -> Unit = {},
    onSectionsClick: () -> Unit = {},
    onReportsClick: () -> Unit = {},
    onAssistClick: () -> Unit = {},
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

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
        modifier = Modifier.fillMaxSize(),
        horizontalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        val syncMessage = when {
            uiState.isLoading -> "Syncing profile and settings..."
            uiState.error != null -> uiState.error
            else -> uiState.syncMessage
        }

        if (syncMessage != null) {
            item {
                val loadIssue = uiState.loadIssue
                val tone = when (loadIssue) {
                    ProfileLoadIssue.OFFLINE -> PegasusRuntimeTone.Offline
                    ProfileLoadIssue.RESTRICTED, ProfileLoadIssue.ERROR -> PegasusRuntimeTone.Warning
                    null -> if (uiState.isLoading) PegasusRuntimeTone.Refreshing else PegasusRuntimeTone.Live
                }
                PegasusRuntimeBanner(
                    tone = tone,
                    message = syncMessage,
                    onRetry = if (!uiState.isLoading) viewModel::refresh else null,
                )
            }
        }

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

        uiState.pricingRulesSummary?.let { summary ->
            item {
                Column(modifier = Modifier.padding(horizontal = PegasusSpacing.lg)) {
                    RetailerSectionHeader(title = stringResource(R.string.mobile_retailer_ui_pricing_rules))
                    Spacer(modifier = Modifier.height(PegasusSpacing.xs))
                    Text(
                        summary,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        uiState.loyaltySummary?.let { summary ->
            item {
                Column(modifier = Modifier.padding(horizontal = PegasusSpacing.lg)) {
                    RetailerSectionHeader(title = "Loyalty")
                    Spacer(modifier = Modifier.height(PegasusSpacing.xs))
                    Text(
                        summary,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        // ── Settings Sections ──
        item {
            SettingsSection(
                onAccountClick = onAccountClick,
                onFamilyMembersClick = onFamilyMembersClick,
                onCapabilitiesClick = onCapabilitiesClick,
                onTeamClick = onTeamClick,
                onLocationsClick = onLocationsClick,
                onStoreStockClick = onStoreStockClick,
                onLocalSkusClick = onLocalSkusClick,
                onPosClick = onPosClick,
                onShiftsClick = onShiftsClick,
                onSectionsClick = onSectionsClick,
                onReportsClick = onReportsClick,
                onAssistClick = onAssistClick,
            )
        }

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

