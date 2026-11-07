package com.pegasusx.factory.ui.components

import androidx.compose.ui.res.stringResource

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import com.pegasus.design.PegasusCollapsibleRail
import com.pegasus.design.PegasusRailGroup
import com.pegasus.design.PegasusRailItem
import com.pegasusx.factory.ui.navigation.FactorySection
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.R

@Composable
fun FactoryNavigationDrawer(
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    selectedRoute: String?,
    onSectionSelected: (FactorySection) -> Unit,
    onSignOut: () -> Unit,
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
            FactorySection.fromRoute(item.id)?.let(onSectionSelected)
                ?: FactorySection.entries.firstOrNull { it.route == item.id }?.let(onSectionSelected)
        },
        modifier = modifier,
        footer = {
            IconButton(onClick = onSignOut) {
                Icon(Icons.AutoMirrored.Filled.ExitToApp, contentDescription = stringResource(R.string.mobile_factory_ui_sign_out))
            }
        },
    )
}

private fun FactorySection.toRailItem() = PegasusRailItem(id = route, label = label, icon = icon)
