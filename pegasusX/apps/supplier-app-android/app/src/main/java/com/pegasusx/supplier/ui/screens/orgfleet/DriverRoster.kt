package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ListItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.data.model.FleetDriver
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.R

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
                    Text(stringResource(R.string.mobile_supplier_ui_nodelabel_phone, nodeLabel(driver.homeNodeType, driver.homeNodeId, topology), driver.phone))
                },
            )
        }
    }
}
