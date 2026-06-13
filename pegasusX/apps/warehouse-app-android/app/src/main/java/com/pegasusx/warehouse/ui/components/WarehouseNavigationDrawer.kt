package com.pegasusx.warehouse.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationDrawerItem
import androidx.compose.material3.NavigationDrawerItemDefaults
import androidx.compose.material3.PermanentDrawerSheet
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.ui.navigation.WarehouseSection
import com.pegasusx.warehouse.ui.theme.PegasusSpacing

@Composable
fun WarehouseNavigationDrawer(
    selectedRoute: String?,
    onSectionSelected: (WarehouseSection) -> Unit,
    onSignOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    PermanentDrawerSheet(
        modifier = modifier
            .width(280.dp)
            .fillMaxHeight(),
        drawerContainerColor = MaterialTheme.colorScheme.surface,
    ) {
        Column(
            modifier = Modifier
                .padding(horizontal = PegasusSpacing.md, vertical = PegasusSpacing.lg),
        ) {
            Text(
                text = "Pegasus Warehouse",
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
            )
            Spacer(Modifier.height(PegasusSpacing.lg))
        }

        Column(
            modifier = Modifier
                .weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = PegasusSpacing.sm),
        ) {
            DrawerGroup("Primary", WarehouseSection.primarySections, selectedRoute, onSectionSelected)
            DrawerGroup("Fulfillment", WarehouseSection.fulfillmentSections, selectedRoute, onSectionSelected)
            DrawerGroup("Inventory", WarehouseSection.inventorySections, selectedRoute, onSectionSelected)
            DrawerGroup("Operations", WarehouseSection.operationsSections, selectedRoute, onSectionSelected)
            DrawerGroup("Portal only", WarehouseSection.portalSections, selectedRoute, onSectionSelected)
        }

        HorizontalDivider(modifier = Modifier.padding(horizontal = PegasusSpacing.lg))
        IconButton(
            onClick = onSignOut,
            modifier = Modifier.padding(PegasusSpacing.lg),
        ) {
            Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = "Sign out")
        }
    }
}

@Composable
private fun DrawerGroup(
    title: String,
    sections: List<WarehouseSection>,
    selectedRoute: String?,
    onSectionSelected: (WarehouseSection) -> Unit,
) {
    Text(
        text = title,
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(horizontal = PegasusSpacing.md, vertical = PegasusSpacing.sm),
    )
    sections.forEach { section ->
        val selected = selectedRoute?.substringBefore("/") == section.route
        NavigationDrawerItem(
            label = { Text(section.label) },
            icon = { Icon(section.icon, contentDescription = section.label) },
            selected = selected,
            onClick = { onSectionSelected(section) },
            colors = NavigationDrawerItemDefaults.colors(
                selectedContainerColor = MaterialTheme.colorScheme.primaryContainer,
                selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                selectedTextColor = MaterialTheme.colorScheme.onPrimaryContainer,
            ),
        )
    }
    Spacer(Modifier.height(PegasusSpacing.sm))
}
