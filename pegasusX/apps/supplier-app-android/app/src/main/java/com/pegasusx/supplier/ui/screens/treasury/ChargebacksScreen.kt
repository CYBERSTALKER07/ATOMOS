package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.TreasuryViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChargebacksScreen(
    onBack: () -> Unit,
    viewModel: TreasuryViewModel = hiltViewModel(),
) {
    val state by viewModel.chargebacksState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Chargebacks") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text("Record chargeback", style = MaterialTheme.typography.titleSmall)
            OutlinedTextField(
                value = state.orderId,
                onValueChange = { viewModel.updateChargebacks { copy(orderId = it) } },
                label = { Text("Order ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.retailerId,
                onValueChange = { viewModel.updateChargebacks { copy(retailerId = it) } },
                label = { Text("Retailer ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.gateway,
                onValueChange = { viewModel.updateChargebacks { copy(gateway = it) } },
                label = { Text("Gateway") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.amount,
                onValueChange = { viewModel.updateChargebacks { copy(amount = it) } },
                label = { Text("Amount (minor units)") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.currency,
                onValueChange = { viewModel.updateChargebacks { copy(currency = it) } },
                label = { Text("Currency") },
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = { viewModel.recordChargeback() },
                enabled = !state.loading,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Record chargeback") }

            HorizontalDivider(Modifier.padding(vertical = PegasusSpacing.sm))
            Text("Record reversal", style = MaterialTheme.typography.titleSmall)
            OutlinedTextField(
                value = state.sessionId,
                onValueChange = { viewModel.updateChargebacks { copy(sessionId = it) } },
                label = { Text("Payment session ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedButton(
                onClick = { viewModel.recordReversal() },
                enabled = !state.loading && state.sessionId.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Record reversal") }

            state.message?.let {
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Live,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            state.error?.let {
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        }
    }
}
