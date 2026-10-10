package com.pegasusx.warehouse.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasus.design.ui.PegasusCollapsibleRail
import com.pegasus.design.ui.PegasusRailGroup
import com.pegasus.design.ui.PegasusRailItem
import com.pegasusx.warehouse.ui.navigation.WarehouseSection
import com.pegasusx.warehouse.R

@Composable
fun WarehouseNavigationDrawer(
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    selectedRoute: String?,
    onSectionSelected: (WarehouseSection) -> Unit,
    onSignOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val selectedId = WarehouseSection.fromRoute(selectedRoute)?.route
    PegasusCollapsibleRail(
        appTitle = "Pegasus Warehouse",
        isExpanded = isExpanded,
        onToggleExpanded = onToggleExpanded,
        groups = warehouseRailGroups(),
        selectedItemId = selectedId,
        onItemSelected = { item ->
            WarehouseSection.fromRoute(item.id)?.let(onSectionSelected)
                ?: WarehouseSection.entries.firstOrNull { it.route == item.id }?.let(onSectionSelected)
        },
        modifier = modifier,
        footer = {
            IconButton(onClick = onSignOut) {
                Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = stringResource(R.string.mobile_warehouse_ui_sign_out))
            }
        },
    )
}

private fun warehouseRailGroups(): List<PegasusRailGroup> = listOf(
    PegasusRailGroup("Primary", WarehouseSection.primarySections.map { it.toRailItem() }),
    PegasusRailGroup("Fulfillment", WarehouseSection.fulfillmentSections.map { it.toRailItem() }),
    PegasusRailGroup("Inventory", WarehouseSection.inventorySections.map { it.toRailItem() }),
    PegasusRailGroup("Operations", WarehouseSection.operationsSections.map { it.toRailItem() }),
    PegasusRailGroup("Settings", WarehouseSection.portalSections.map { it.toRailItem() }),
)

private fun WarehouseSection.toRailItem() = PegasusRailItem(id = route, label = label, icon = icon)
