package com.pegasusx.supplier.ui.screens.profile

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierProfile
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.R

@Composable
fun SupplierIdentityCard(p: SupplierProfile, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        Text(p.legalName, style = MaterialTheme.typography.headlineSmall)
        Text(p.contactName)
        Text(p.email)
        Text(p.phone)
        Text(stringResource(R.string.mobile_supplier_ui_country_currency, p.country, p.currency))
        Text(stringResource(R.string.mobile_supplier_ui_configured_isconfigured, p.isConfigured))
    }
}
