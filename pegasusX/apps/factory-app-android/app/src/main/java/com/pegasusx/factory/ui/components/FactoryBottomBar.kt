package com.pegasusx.factory.ui.components

import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.factory.ui.navigation.FactorySection

@Composable
fun FactoryBottomBar(
    selectedRoute: String?,
    onSectionSelected: (FactorySection) -> Unit,
    modifier: Modifier = Modifier,
) {
    val active = FactorySection.fromRoute(selectedRoute)
    NavigationBar(modifier = modifier) {
        FactorySection.compactTabs.forEach { section ->
            val selected = when (section) {
                FactorySection.MORE -> active != null && active !in FactorySection.compactTabs.filter { it != FactorySection.MORE }
                else -> active == section
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
