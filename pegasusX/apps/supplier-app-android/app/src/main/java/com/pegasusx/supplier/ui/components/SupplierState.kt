package com.pegasusx.supplier.ui.components

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane

typealias SupplierStateKind = PegasusStateKind
typealias SupplierRuntimeTone = PegasusRuntimeTone

@Composable
fun SupplierLoadingState(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
) = PegasusLoadingState(title, body, modifier)

@Composable
fun SupplierStatePane(
    kind: SupplierStateKind,
    headline: String,
    body: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) = PegasusStatePane(kind, headline, body, modifier, actionLabel, onAction)

@Composable
fun SupplierRuntimeBanner(
    tone: SupplierRuntimeTone,
    message: String,
    modifier: Modifier = Modifier,
) = PegasusRuntimeBanner(tone, message, modifier)
