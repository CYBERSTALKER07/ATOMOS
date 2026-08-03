package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ListItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.FleetDriver
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun DriverRoster(drivers: List<FleetDriver>, topology: SupplierTopologyResponse?) {
    if (drivers.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No drivers", "Create a driver to start fleet onboarding.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(drivers, key = { it.driverId }) { driver ->
            ListItem(
                headlineContent = { Text(driver.name) },
                supportingContent = {
                    Text("${nodeLabel(driver.homeNodeType, driver.homeNodeId, topology)} · ${driver.phone}")
                },
            )
        }
    }
}
