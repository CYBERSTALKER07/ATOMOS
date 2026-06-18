package com.pegasusx.factory.ui.components

import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.navigation.FactoryRoutes
import com.pegasusx.factory.ui.navigation.FactorySection

@Composable
fun FactoryBottomBar(
    selectedRoute: String?,
    onSectionSelected: (FactorySection) -> Unit,
    modifier: Modifier = Modifier,
) {
    NavigationBar(modifier = modifier) {
        FactorySection.compactTabs.forEach { section ->
            val selected = when {
                section == FactorySection.MORE -> selectedRoute?.substringBefore("/") in
                    FactorySection.operationsSections.map { it.route } + FactorySection.intelligenceSections.map { it.route } + FactoryRoutes.MORE
                else -> selectedRoute?.substringBefore("/") == section.route
            }
            NavigationBarItem(
                selected = selected,
                onClick = { onSectionSelected(section) },
                icon = { Icon(section.icon, contentDescription = section.label) },
                label = { Text(section.label) },
            )
        }
    }
}
