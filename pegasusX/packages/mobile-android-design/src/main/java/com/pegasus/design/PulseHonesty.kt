package com.pegasus.design

/** Fail-closed mapping for native pulse GET. HTTP failure is not an empty timeline. */
object PulseHonesty {
    const val FAILED = "pulse_failed"
    const val COMMAND_FAILED = "control_tower_pulse_failed"

    data class Result<T>(val events: List<T>, val error: String?)
    data class ObjectResult<T>(val value: T?, val error: String?)

    fun <T> applyHttp(ok: Boolean, incoming: List<T>?, previous: List<T>): Result<T> {
        return if (ok && incoming != null) Result(incoming, null)
        else Result(previous, FAILED)
    }

    fun <T> applyObject(ok: Boolean, incoming: T?, previous: T?): ObjectResult<T> {
        return if (ok && incoming != null) ObjectResult(incoming, null)
        else ObjectResult(previous, COMMAND_FAILED)
    }
}
