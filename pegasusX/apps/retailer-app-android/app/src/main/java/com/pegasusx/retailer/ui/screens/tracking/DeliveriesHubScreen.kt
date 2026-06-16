package com.pegasusx.retailer.ui.screens.tracking

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.screens.dock.DockScreen
import com.pegasusx.retailer.ui.screens.dock.DockViewModel

private enum class DeliveriesTab(val label: String) {
    Map("Map"),
    Dock("Dock Queue"),
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeliveriesHubScreen(
    initialTabIndex: Int = 0,
    mapViewModel: DeliveryTrackingViewModel = hiltViewModel(),
    dockViewModel: DockViewModel = hiltViewModel(),
) {
    var selectedTab by rememberSaveable { mutableIntStateOf(initialTabIndex.coerceIn(0, DeliveriesTab.entries.lastIndex)) }

    Column(modifier = Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text("Deliveries") },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.surface,
            ),
        )

        PrimaryTabRow(selectedTabIndex = selectedTab) {
            DeliveriesTab.entries.forEachIndexed { index, tab ->
                Tab(
                    selected = selectedTab == index,
                    onClick = { selectedTab = index },
                    text = { Text(tab.label) },
                )
            }
        }

        when (DeliveriesTab.entries[selectedTab]) {
            DeliveriesTab.Map -> DeliveryMapScreen(
                viewModel = mapViewModel,
                onBack = {},
                embedded = true,
                modifier = Modifier.fillMaxSize(),
            )
            DeliveriesTab.Dock -> DockScreen(
                viewModel = dockViewModel,
                modifier = Modifier.fillMaxSize(),
            )
        }
    }
}
