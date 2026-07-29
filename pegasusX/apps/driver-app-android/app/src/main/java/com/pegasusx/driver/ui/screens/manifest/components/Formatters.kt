package com.pegasusx.driver.ui.screens.manifest.components

import java.text.NumberFormat
import java.util.Locale

val amountFormatter: NumberFormat by lazy {
    NumberFormat.getNumberInstance(Locale.US).apply {
        maximumFractionDigits = 0
        isGroupingUsed = true
    }
}

fun Long.formatAmount(): String = "${amountFormatter.format(this)}"
