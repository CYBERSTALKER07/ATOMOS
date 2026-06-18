package com.pegasusx.warehouse.ui.components

import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.ui.navigation.WarehouseSection

@Composable
fun WarehouseBottomBar(
    selectedRoute: String?,
    onSectionSelected: (WarehouseSection) -> Unit,
    modifier: Modifier = Modifier,
) {
    NavigationBar(modifier = modifier) {
        WarehouseSection.compactTabs.forEach { section ->
            val selected = WarehouseSection.fromRoute(selectedRoute) == section
            NavigationBarItem(
                selected = selected,
                onClick = { onSectionSelected(section) },
                icon = { Icon(section.icon, contentDescription = section.label) },
                label = { Text(section.label) },
            )
        }
    }
}
