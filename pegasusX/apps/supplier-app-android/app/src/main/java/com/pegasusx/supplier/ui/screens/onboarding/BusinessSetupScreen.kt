package com.pegasusx.supplier.ui.screens.onboarding

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasusx.supplier.ui.design.PegasusRuntimeBanner
import com.pegasusx.supplier.ui.design.PegasusRuntimeTone
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.OnboardingViewModel
import com.pegasusx.supplier.R

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
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
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
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OutlinedTextField(
                    value = state.taxId,
                    onValueChange = { next ->
                        viewModel.updateBusiness {
                            copy(
                                taxId = next,
                                innLookup = null,
                                innError = null,
                                joinRequestConflict = null,
                                joinRequestSent = false,
                                error = null,
                            )
                        }
                    },
                    label = { Text(stringResource(R.string.supplier_portal_onboarding_inn_tax_id_label)) },
                    placeholder = { Text(stringResource(R.string.supplier_portal_onboarding_inn_tax_id_placeholder)) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                )
                OutlinedButton(
                    onClick = { viewModel.verifyINN() },
                    enabled = !state.innChecking && !state.loading,
                ) {
                    Text(
                        if (state.innChecking) {
                            stringResource(R.string.supplier_portal_onboarding_inn_verifying)
                        } else {
                            stringResource(R.string.supplier_portal_onboarding_inn_verify_button)
                        },
                    )
                }
            }
            state.innLookup?.let { lookup ->
                if (!lookup.legalName.isNullOrBlank()) {
                    OutlinedCard(
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(
                            modifier = Modifier.padding(PegasusSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(4.dp),
                        ) {
                            Text(
                                text = stringResource(R.string.supplier_portal_onboarding_inn_verified_badge),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.primary,
                                fontWeight = FontWeight.Bold,
                            )
                            Text(
                                text = lookup.legalName,
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                            )
                            lookup.address?.takeIf { it.isNotBlank() }?.let { addr ->
                                Text(addr, style = MaterialTheme.typography.bodySmall)
                            }
                            if (lookup.existingJoinEligible) {
                                Text(
                                    text = stringResource(R.string.supplier_portal_onboarding_inn_existing_company_message),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.error,
                                )
                            }
                        }
                    }
                }
            }
            state.innError?.let { innErr ->
                PegasusRuntimeBanner(
                    tone = PegasusRuntimeTone.Warning,
                    message = innErr,
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            if (state.joinRequestSent) {
                OutlinedCard(
                    shape = RoundedCornerShape(12.dp),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Column(
                        modifier = Modifier.padding(PegasusSpacing.md),
                        verticalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        Text(
                            text = stringResource(R.string.supplier_portal_onboarding_inn_existing_company_title),
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            text = stringResource(R.string.supplier_portal_onboarding_inn_join_request_sent),
                            style = MaterialTheme.typography.bodySmall,
                        )
                        state.joinRequestConflict?.legalName?.let { name ->
                            Text(
                                text = "$name (${state.joinRequestConflict?.normalizedInn.orEmpty()})",
                                style = MaterialTheme.typography.labelMedium,
                                fontFamily = FontFamily.Monospace,
                            )
                        }
                    }
                }
            }
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
