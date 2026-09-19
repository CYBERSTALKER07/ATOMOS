package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ListItem
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.data.model.SupplierOrgMember
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun OrgRoster(
    members: List<SupplierOrgMember>,
    onEdit: (SupplierOrgMember) -> Unit,
    onDeactivate: (String) -> Unit,
    actionId: String?,
) {
    if (members.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No org members", "Create warehouse, factory, or payload staff.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(members, key = { it.userId }) { member ->
            ListItem(
                headlineContent = { Text(member.name) },
                supportingContent = {
                    Text("${member.supplierRole} · ${member.phone} · ${if (member.isActive) "Active" else "Inactive"}")
                },
                trailingContent = {
                    Row {
                        TextButton(
                            enabled = actionId != member.userId,
                            onClick = { onEdit(member) },
                        ) { Text("Edit") }
                        if (member.isActive) {
                            TextButton(
                                enabled = actionId != member.userId,
                                onClick = { onDeactivate(member.userId) },
                            ) { Text("Deactivate") }
                        }
                    }
                },
            )
        }
    }
}
