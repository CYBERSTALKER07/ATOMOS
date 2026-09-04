package com.pegasusx.retailer.ui.screens.orders.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.RetailerAIPrediction
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.retailer.ui.components.ShimmerOrderList

@Composable
fun ActiveOrdersList(
    orders: List<Order>,
    isLoading: Boolean = false,
    onDetailsCash: (Order) -> Unit,
    onQRCash: (Order) -> Unit,
) {
    if (isLoading && orders.isEmpty()) {
        ShimmerOrderList()
        return
    }
    if (orders.isEmpty()) {
        PegasusStatePane(kind = PegasusStateKind.Empty, headline = "No Active Orders", body = "Orders being prepared or en route will appear here")
        return
    }
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        itemsIndexed(orders, key = { _, o -> o.id }) { _, order ->
            ActiveOrderCard(
                order = order,
                onDetailsCash = { onDetailsCash(order) },
                onQRCash = { onQRCash(order) },
            )
        }
        item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
    }
}

@Composable
fun OrderedList(
    orders: List<Order>,
    isLoading: Boolean = false,
    onDetailsCash: (Order) -> Unit,
    onCancel: (Order) -> Unit,
    onConfirmAi: (Order) -> Unit = {},
    onRejectAi: (Order) -> Unit = {},
    onConfirmPreorder: (Order) -> Unit = {},
    onEditPreorder: (Order) -> Unit = {},
    onAcceptDeliveryProposal: (Order) -> Unit = {},
    onRejectDeliveryProposal: (Order) -> Unit = {},
) {
    if (isLoading && orders.isEmpty()) {
        ShimmerOrderList()
        return
    }
    if (orders.isEmpty()) {
        PegasusStatePane(kind = PegasusStateKind.Empty, headline = "No Pending Orders", body = "Orders awaiting dispatch will appear here")
        return
    }
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        itemsIndexed(orders, key = { _, o -> o.id }) { _, order ->
            OrderedCard(
                order = order,
                onDetailsCash = { onDetailsCash(order) },
                onCancel = { onCancel(order) },
                onConfirmAi = { onConfirmAi(order) },
                onRejectAi = { onRejectAi(order) },
                onConfirmPreorder = { onConfirmPreorder(order) },
                onEditPreorder = { onEditPreorder(order) },
                onAcceptDeliveryProposal = { onAcceptDeliveryProposal(order) },
                onRejectDeliveryProposal = { onRejectDeliveryProposal(order) },
            )
        }
        item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
    }
}

@Composable
fun AiPlannedList(
    predictions: List<RetailerAIPrediction>,
    isLoading: Boolean = false,
    onConfirm: (RetailerAIPrediction) -> Unit,
    onReject: (RetailerAIPrediction) -> Unit,
) {
    if (isLoading && predictions.isEmpty()) {
        ShimmerOrderList(count = 3)
        return
    }
    if (predictions.isEmpty()) {
        PegasusStatePane(kind = PegasusStateKind.Empty, headline = "No pending AI preorders", body = "AI restock preorders waiting for confirm or reject will appear here")
        return
    }
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        itemsIndexed(predictions, key = { _, f -> f.orderId }) { _, item ->
            AiPlannedCard(
                item = item,
                onConfirm = { onConfirm(item) },
                onReject = { onReject(item) },
            )
        }
        item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
    }
}
