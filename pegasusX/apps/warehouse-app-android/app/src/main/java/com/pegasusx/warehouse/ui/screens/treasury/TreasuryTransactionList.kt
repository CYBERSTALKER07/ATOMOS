package com.pegasusx.warehouse.ui.screens.treasury

<<<<<<< HEAD
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
=======
import androidx.compose.foundation.layout.*
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
<<<<<<< HEAD
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.warehouse.data.model.Invoice
=======
import com.pegasusx.warehouse.data.model.Invoice
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
import com.pegasusx.warehouse.ui.components.WarehouseOpsListCard
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import java.text.NumberFormat

@Composable
fun TreasuryTransactionList(
    invoices: List<Invoice>,
<<<<<<< HEAD
    fmt: NumberFormat,
    modifier: Modifier = Modifier
=======
    fmt: NumberFormat
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
) {
    if (invoices.isEmpty()) {
        PegasusStatePane(
            kind = PegasusStateKind.Empty,
            headline = "No invoices",
            body = "Retailer invoices will appear here when issued.",
<<<<<<< HEAD
            modifier = modifier
=======
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
        )
    } else {
        LazyVerticalGrid(
            columns = GridCells.Adaptive(minSize = 340.dp),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
<<<<<<< HEAD
            modifier = modifier
=======
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
        ) {
            items(invoices, key = { it.invoiceId }) { inv ->
                val displayAmount = if (inv.amount > 0) inv.amount else inv.amountUzs
                val displayCurrency = if (inv.currency.isBlank()) "UZS" else inv.currency.uppercase()
                val payoutOwner = buildString {
                    append(if (inv.payoutOwnerType.isBlank()) "SUPPLIER" else inv.payoutOwnerType)
                    if (inv.payoutOwnerId.isNotBlank()) {
                        append(":")
                        append(inv.payoutOwnerId.take(8))
                    }
                }
                WarehouseOpsListCard(
                    headline = inv.retailerName,
                    supporting = "${fmt.format(displayAmount)} $displayCurrency · due ${inv.dueDate} · Owner $payoutOwner · Net ${fmt.format(inv.netPayoutAmount)}",
                    status = inv.status,
                )
            }
        }
    }
}
