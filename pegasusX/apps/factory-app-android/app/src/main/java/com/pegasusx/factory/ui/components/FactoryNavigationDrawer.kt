package com.pegasusx.factory.ui.components

import com.pegasus.design.PegasusCollapsibleRail
import com.pegasus.design.PegasusRailGroup
import com.pegasus.design.PegasusRailItem
import com.pegasusx.factory.ui.navigation.FactorySection
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier

@Composable
fun FactoryNavigationDrawer(
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    selectedRoute: String?,
    onSectionSelected: (FactorySection) -> Unit,
    modifier: Modifier = Modifier,
) {
    val selectedId = FactorySection.fromRoute(selectedRoute)?.route
    PegasusCollapsibleRail(
        appTitle = "Pegasus Factory",
        isExpanded = isExpanded,
        onToggleExpanded = onToggleExpanded,
        groups = listOf(
            PegasusRailGroup("Primary", FactorySection.primarySections.map { it.toRailItem() }),
            PegasusRailGroup("Operations", FactorySection.operationsSections.map { it.toRailItem() }),
            PegasusRailGroup("Intelligence", FactorySection.intelligenceSections.map { it.toRailItem() }),
        ),
        selectedItemId = selectedId,
        onItemSelected = { item ->
            FactorySection.entries.firstOrNull { it.route == item.id }?.let(onSectionSelected)
        },
        modifier = modifier,
    )
}

private fun FactorySection.toRailItem() = PegasusRailItem(id = route, label = label, icon = icon)
