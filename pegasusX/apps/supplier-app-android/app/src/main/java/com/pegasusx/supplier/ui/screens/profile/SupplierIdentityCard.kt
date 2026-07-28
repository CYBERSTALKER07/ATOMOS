package com.pegasusx.supplier.ui.screens.profile

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierProfile
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun SupplierIdentityCard(p: SupplierProfile, modifier: Modifier = Modifier) {
    Column(modifier, verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
        Text(p.legalName, style = MaterialTheme.typography.headlineSmall)
        Text(p.contactName)
        Text(p.email)
        Text(p.phone)
        Text("${p.country} · ${p.currency}")
        Text("Configured: ${p.isConfigured}")
    }
}
