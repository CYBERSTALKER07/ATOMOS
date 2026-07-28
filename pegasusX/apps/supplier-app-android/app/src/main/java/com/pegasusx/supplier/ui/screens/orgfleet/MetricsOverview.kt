package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun MetricsOverview(
    driversCount: Int,
    vehiclesCount: Int,
    orgMembersCount: Int,
    topology: SupplierTopologyResponse?
) {
    LazyRow(
        contentPadding = PaddingValues(horizontal = PegasusSpacing.lg),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = Modifier.padding(top = PegasusSpacing.md)
    ) {
        item { MetricCard("Warehouses", topology?.warehouses?.size ?: 0) }
        item { MetricCard("Factories", topology?.factories?.size ?: 0) }
        item { MetricCard("Org members", orgMembersCount) }
        item { MetricCard("Fleet entities", driversCount + vehiclesCount) }
    }
}

@Composable
private fun MetricCard(title: String, count: Int) {
    Column(
        modifier = Modifier
            .widthIn(min = 120.dp)
            .background(
                MaterialTheme.colorScheme.surfaceVariant,
                RoundedCornerShape(8.dp)
            )
            .padding(PegasusSpacing.md)
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = count.toString(),
            style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
