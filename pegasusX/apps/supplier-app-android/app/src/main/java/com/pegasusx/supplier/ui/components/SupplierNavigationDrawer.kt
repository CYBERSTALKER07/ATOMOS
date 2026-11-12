package com.pegasusx.supplier.ui.components

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.ui.design.PegasusCollapsibleRail
import com.pegasusx.supplier.ui.design.PegasusRailGroup
import com.pegasusx.supplier.ui.design.PegasusRailItem
import com.pegasusx.supplier.ui.navigation.SupplierSection

@Composable
fun SupplierNavigationDrawer(
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    selectedRoute: String?,
    onSectionSelected: (SupplierSection) -> Unit,
    modifier: Modifier = Modifier,
) {
    PegasusCollapsibleRail(
        appTitle = "Pegasus Supplier",
        isExpanded = isExpanded,
        onToggleExpanded = onToggleExpanded,
        groups = listOf(
            PegasusRailGroup("Primary", SupplierSection.compactTabs.filter { it != SupplierSection.MORE }.map { it.toRailItem() }),
            PegasusRailGroup("Operations", SupplierSection.opsSections.map { it.toRailItem() }),
            PegasusRailGroup("Intelligence", SupplierSection.intelligenceSections.map { it.toRailItem() }),
            PegasusRailGroup("Network", SupplierSection.networkSections.map { it.toRailItem() }),
            PegasusRailGroup("Account", SupplierSection.accountSections.map { it.toRailItem() }),
        ),
        selectedItemId = SupplierSection.fromRoute(selectedRoute)?.route,
        onItemSelected = { item ->
            SupplierSection.entries.firstOrNull { it.route == item.id }?.let(onSectionSelected)
        },
        modifier = modifier,
    )
}

private fun SupplierSection.toRailItem() = PegasusRailItem(id = route, label = label, icon = icon)
