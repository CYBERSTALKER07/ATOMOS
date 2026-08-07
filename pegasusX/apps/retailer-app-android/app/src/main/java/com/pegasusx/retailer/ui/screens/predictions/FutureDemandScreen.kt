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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.ui.screens.orders.components.AiPlannedCard
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.retailer.ui.screens.orders.OrdersViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FutureDemandScreen(
    onBack: () -> Unit = {},
    viewModel: OrdersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var correctionForecast by remember { mutableStateOf<DemandForecast?>(null) }
    var correctionAmount by remember { mutableStateOf("") }

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
                    message = "Fetching AI demand forecasts for your store.",
                )
            }
        } else if (uiState.predictions.isEmpty()) {
            PegasusEmptyState(
                icon = Icons.Rounded.AutoAwesome,
                title = stringResource(R.string.mobile_retailer_ui_no_ai_predictions),
                message = "AI-predicted orders based on your history will appear here.",
            )
        } else {
            LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            
    ) {
                itemsIndexed(uiState.predictions, key = { _, f -> f.id }) { _, forecast ->
                    AiPlannedCard(
                        forecast = forecast,
                        onPreorder = { viewModel.requestPreorder(forecast) },
                        onCorrect = {
                            correctionForecast = forecast
                            correctionAmount = forecast.predictedQuantity.toString()
                        },
                        onReject = { viewModel.rejectPrediction(forecast.id) },
                    )
                    Spacer(modifier = Modifier.height(12.dp))
                }
                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
            }
        }
    }

    correctionForecast?.let { forecast ->
        AlertDialog(
            onDismissRequest = { correctionForecast = null },
            title = { Text("Correct Prediction") },
            text = {
                Column {
                    Text(stringResource(R.string.mobile_retailer_ui_productname_ai_predicted_predictedquantity_units, forecast.productName, forecast.predictedQuantity))
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = correctionAmount,
                        onValueChange = { correctionAmount = it.filter { ch -> ch.isDigit() } },
                        label = { Text("Correct amount") },
                        singleLine = true,
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        correctionAmount.toLongOrNull()?.let { amount ->
                            viewModel.correctPrediction(forecast.id, amount)
                        }
                        correctionForecast = null
                    },
                ) { Text("Submit", fontWeight = FontWeight.Bold) }
            },
            dismissButton = {
                TextButton(
                    onClick = {
                        viewModel.rejectPrediction(forecast.id)
                        correctionForecast = null
                    },
                ) { Text("Reject") }
            },
        )
    }
}
