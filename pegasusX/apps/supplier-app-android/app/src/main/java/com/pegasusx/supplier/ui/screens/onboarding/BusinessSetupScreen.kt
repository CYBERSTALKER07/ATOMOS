package com.pegasusx.supplier.ui.screens.onboarding

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.OnboardingViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BusinessSetupScreen(
    onComplete: () -> Unit,
    onBack: (() -> Unit)? = null,
    viewModel: OnboardingViewModel = hiltViewModel(),
) {
    val state by viewModel.businessState.collectAsStateWithLifecycle()
    val fromOnboarding = onBack == null

    Scaffold(topBar = {
        TopAppBar(
            title = { Text("Business setup") },
            navigationIcon = {
                if (onBack != null) {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                }
            },
        )
    }) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Enter tax and headquarters details to complete supplier registration.",
                style = MaterialTheme.typography.bodyMedium,
            )
            OutlinedTextField(
                value = state.taxId,
                onValueChange = { viewModel.updateBusiness { copy(taxId = it) } },
                label = { Text("Tax ID") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.registrationNumber,
                onValueChange = { viewModel.updateBusiness { copy(registrationNumber = it) } },
                label = { Text("Registration number") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.headquartersAddress,
                onValueChange = { viewModel.updateBusiness { copy(headquartersAddress = it) } },
                label = { Text("Headquarters address") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.city,
                onValueChange = { viewModel.updateBusiness { copy(city = it) } },
                label = { Text("City") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.postalCode,
                onValueChange = { viewModel.updateBusiness { copy(postalCode = it) } },
                label = { Text("Postal code") },
                modifier = Modifier.fillMaxWidth(),
            )
            state.error?.let {
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = it,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            Button(
                onClick = { viewModel.submitBusinessSetup(onComplete) },
                enabled = !state.loading && state.taxId.isNotBlank() &&
                    state.headquartersAddress.isNotBlank() && state.city.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (state.loading) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Text(if (fromOnboarding) "Continue to billing" else "Save")
                }
            }
        }
    }
}
