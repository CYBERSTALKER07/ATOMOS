package com.pegasusx.retailer.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.outlined.CreditCard
import androidx.compose.material.icons.outlined.Layers
import androidx.compose.material.icons.outlined.Category
import androidx.compose.material.icons.outlined.LocalMall
import androidx.compose.material.icons.outlined.LocationOn
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material.icons.outlined.GridView
import androidx.compose.material.icons.outlined.SupportAgent
import androidx.compose.material.icons.outlined.Assessment
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.People
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.R

@Composable
fun SettingsSection(
    onAccountClick: () -> Unit,
    onSavedCardsClick: () -> Unit,
    onFamilyMembersClick: () -> Unit,
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
) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(3.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column {
            SettingsListItem(icon = Icons.Outlined.Settings, title = stringResource(R.string.mobile_retailer_ui_general_settings), subtitle = "Language, preferences", onClick = { })
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Rounded.Person, title = stringResource(R.string.supplier_portal_auth_register_steps_account), subtitle = "Business details & receiving hours", onClick = onAccountClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.Layers, title = stringResource(R.string.retailer_desktop_settings_capabilities_text_store_capabilities), subtitle = "Team, stock, POS packs", onClick = onCapabilitiesClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.LocationOn, title = stringResource(R.string.retailer_desktop_settings_locations_text_locations), subtitle = "Branches and checkout store", onClick = onLocationsClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.LocalMall, title = stringResource(R.string.portal_nav_store_stock), subtitle = "Receive, putaway, count", onClick = onStoreStockClick)
            SettingsListItem(icon = Icons.Outlined.Category, title = stringResource(R.string.portal_nav_local_skus), subtitle = "Non-Pegasus POS goods", onClick = onLocalSkusClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.ShoppingCart, title = "POS", subtitle = "Cashier sales and voids", onClick = onPosClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.Schedule, title = stringResource(R.string.portal_nav_shifts), subtitle = "Clock in and cash recon", onClick = onShiftsClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.GridView, title = stringResource(R.string.portal_nav_sections), subtitle = "Departments and SKU map", onClick = onSectionsClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.Assessment, title = stringResource(R.string.portal_nav_reports_pro), subtitle = "Sales and inventory digest", onClick = onReportsClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.SupportAgent, title = stringResource(R.string.retailer_desktop_assist_text_floor_assist), subtitle = "Section help tickets", onClick = onAssistClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.People, title = stringResource(R.string.retailer_desktop_settings_team_text_team), subtitle = "Staff roles and invites", onClick = onTeamClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.CreditCard, title = stringResource(R.string.retailer_desktop_settings_cards_text_saved_cards), subtitle = "Manage payment methods", onClick = onSavedCardsClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.People, title = stringResource(R.string.mobile_retailer_ui_family_contacts), subtitle = "Legacy name/phone list", onClick = onFamilyMembersClick)
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.15f), modifier = Modifier.padding(horizontal = 16.dp))
            SettingsListItem(icon = Icons.Outlined.Notifications, title = stringResource(R.string.portal_nav_notifications), subtitle = "Push, email, SMS", onClick = { })
        }
    }
}

@Composable
fun SettingsListItem(icon: ImageVector, title: String, subtitle: String, onClick: () -> Unit) {
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
