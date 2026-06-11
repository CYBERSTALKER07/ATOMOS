package com.pegasusx.driver.util

import java.security.MessageDigest

/** Returns the SHA-256 hex digest of a UTF-8 string (mirrors iOS sha256Hex). */
fun sha256Hex(input: String): String {
    val digest = MessageDigest.getInstance("SHA-256").digest(input.toByteArray(Charsets.UTF_8))
    return digest.joinToString("") { byte -> "%02x".format(byte) }
}
