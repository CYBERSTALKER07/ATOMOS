package com.pegasusx.retailer.ui.screens.profile

internal object ReceivingWindowValidator {
    private val pattern = Regex("^([01]?\\d|2[0-3]):([0-5]?\\d)$")

    fun normalize(raw: String): String {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return ""
        val match = pattern.matchEntire(trimmed) ?: return trimmed
        val hour = match.groupValues[1].padStart(2, '0')
        val minute = match.groupValues[2].padStart(2, '0')
        return "$hour:$minute"
    }

    fun validate(raw: String): String? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null
        val normalized = normalize(trimmed)
        return if (pattern.matches(normalized)) {
            null
        } else {
            "Use 24-hour HH:MM format (e.g. 09:00)"
        }
    }
}
