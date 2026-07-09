package com.pegasusx.factory.ui.screens.more

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.ui.navigation.FactorySection

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FactoryMoreHubScreen(
    onNavigate: (String) -> Unit,
) {
    val primaryOverflow = FactorySection.primarySections.filter {
        it !in FactorySection.compactTabs.filter { tab -> tab != FactorySection.MORE }
    }

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("More") })
        },
    ) { padding ->
        LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        
    ) {
            if (primaryOverflow.isNotEmpty()) {
                item(span = { GridItemSpan(maxLineSpan) }) { SectionHeader("Primary") }
                primaryOverflow.forEach { section ->
                    item { MoreRow(section) { onNavigate(section.route) } }
                }
            }
            item(span = { GridItemSpan(maxLineSpan) }) { SectionHeader("Operations") }
            FactorySection.operationsSections.forEach { section ->
                item { MoreRow(section) { onNavigate(section.route) } }
            }
            item(span = { GridItemSpan(maxLineSpan) }) { SectionHeader("Intelligence") }
            FactorySection.intelligenceSections.forEach { section ->
                item { MoreRow(section) { onNavigate(section.route) } }
            }
        }
    }
}

@Composable
private fun SectionHeader(title: String) {
    Text(
        text = title.uppercase(),
        modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
        style = MaterialTheme.typography.labelSmall,
        color = MaterialTheme.colorScheme.primary,
    )
}

@Composable
private fun MoreRow(section: FactorySection, onClick: () -> Unit) {
    ListItem(
        headlineContent = { Text(section.label) },
        leadingContent = { Icon(section.icon, contentDescription = section.label) },
        modifier = Modifier.clickable(onClick = onClick),
    )
    HorizontalDivider()
}
