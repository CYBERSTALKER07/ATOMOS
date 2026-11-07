package com.pegasusx.retailer.ui.screens.predictions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.itemsIndexed

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.screens.orders.components.AiPlannedCard
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.retailer.ui.screens.orders.OrdersViewModel
import com.pegasusx.retailer.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FutureDemandScreen(
    onBack: () -> Unit = {},
    viewModel: OrdersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    Column(modifier = Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text("Reorder suggestions") },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                }
            },
            colors = TopAppBarDefaults.topAppBarColors(
                containerColor = MaterialTheme.colorScheme.surface,
            ),
        )

        uiState.syncMessage?.let { message ->
            PegasusRuntimeBanner(
                tone = PegasusRuntimeTone.Warning,
                message = message,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                onRetry = viewModel::refresh,
            )
        }

        if (uiState.isLoading && uiState.predictions.isEmpty()) {
            Box(modifier = Modifier.fillMaxSize()) {
                PegasusEmptyState(
                    icon = Icons.Rounded.AutoAwesome,
                    title = stringResource(R.string.mobile_retailer_ui_loading_predictions),
                    message = "Fetching pending AI preorders for your store.",
                )
            }
        } else if (uiState.predictions.isEmpty()) {
            PegasusEmptyState(
                icon = Icons.Rounded.AutoAwesome,
                title = stringResource(R.string.mobile_retailer_ui_no_ai_predictions),
                message = "Pending AI preorders waiting for confirm or reject will appear here.",
            )
        } else {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            ) {
                itemsIndexed(uiState.predictions, key = { _, item -> item.orderId }) { _, item ->
                    AiPlannedCard(
                        item = item,
                        onConfirm = { viewModel.confirmAiOrder(item.orderId) },
                        onReject = { viewModel.rejectAiOrder(item.orderId) },
                    )
                    Spacer(modifier = Modifier.height(12.dp))
                }
                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
            }
        }
    }
}
