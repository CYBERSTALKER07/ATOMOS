package com.pegasus.barcode

/** Normalizes retail EAN/GTIN input (digits only, valid length + checksum). */
object EanBarcode {
    fun normalize(raw: String): String? {
        val digits = raw.filter { it.isDigit() }
        if (digits.isEmpty()) return null
        if (digits.length !in setOf(8, 12, 13, 14)) return null
        if (!validGtinChecksum(digits)) return null
        return digits
    }

    private fun validGtinChecksum(code: String): Boolean {
        val n = code.length
        if (n < 8) return false
        var sum = 0
        for (i in 0 until n - 1) {
            val d = code[i].digitToIntOrNull() ?: return false
            val posFromRight = n - 1 - i
            sum += if (posFromRight % 2 == 1) d * 3 else d
        }
        val check = code.last().digitToIntOrNull() ?: return false
        return (10 - (sum % 10)) % 10 == check
    }
}
