package com.pegasusx.factory.ui.screens.staff.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.factory.data.model.StaffMember
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasusx.factory.ui.theme.PegasusSpacing
import com.pegasusx.factory.R

@Composable
fun StaffList(
    staff: List<StaffMember>,
    onStaffClick: (String) -> Unit,
    onShift: Int,
    innerPadding: PaddingValues
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
        item {
            StaffSummaryCard(
                total = staff.size,
                onShift = onShift,
            )
        }
        item {
            FactorySectionTitle(title = stringResource(R.string.mobile_factory_ui_operator_roster))
        }
        items(staff, key = { it.id }) { member ->
            FactoryOpsListCard(
                headline = member.name,
                supporting = "${member.role} · ${member.phone}",
                status = member.status,
                onClick = { onStaffClick(member.id) },
            )
        }
    }
}
