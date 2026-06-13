package com.pegasusx.driver.data.telemetry

import android.app.Application
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import android.speech.tts.TextToSpeech
import android.speech.tts.UtteranceProgressListener
import java.util.Locale

class NavigationCueAnnouncer(
    private val app: Application,
) {
    private var textToSpeech: TextToSpeech? = null
    private var isReady = false
    private var pendingInstruction: String? = null

    fun onManeuverAdvanced(cue: NavigationCue) {
        vibrateManeuver()
        speakInstruction(cue.instruction)
    }

    fun stop() {
        pendingInstruction = null
        textToSpeech?.stop()
    }

    fun shutdown() {
        stop()
        textToSpeech?.shutdown()
        textToSpeech = null
        isReady = false
    }

    private fun speakInstruction(instruction: String) {
        val trimmed = instruction.trim()
        if (trimmed.isEmpty()) {
            return
        }
        val tts = textToSpeech
        if (tts == null || !isReady) {
            pendingInstruction = trimmed
            ensureTts()
            return
        }
        pendingInstruction = null
        tts.speak(trimmed, TextToSpeech.QUEUE_FLUSH, null, "nav-maneuver")
    }

    private fun ensureTts() {
        if (textToSpeech != null) {
            return
        }
        textToSpeech = TextToSpeech(app) { status ->
            isReady = status == TextToSpeech.SUCCESS
            if (!isReady) {
                return@TextToSpeech
            }
            textToSpeech?.language = Locale.getDefault()
            textToSpeech?.setOnUtteranceProgressListener(object : UtteranceProgressListener() {
                override fun onStart(utteranceId: String?) = Unit

                override fun onDone(utteranceId: String?) = Unit

                @Deprecated("Deprecated in Java")
                override fun onError(utteranceId: String?) = Unit
            })
            pendingInstruction?.let(::speakInstruction)
        }
    }

    private fun vibrateManeuver() {
        val vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            app.getSystemService(VibratorManager::class.java)?.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            app.getSystemService(Vibrator::class.java)
        } ?: return
        val pattern = longArrayOf(0, 70, 50, 70)
        val amplitudes = intArrayOf(0, VibrationEffect.DEFAULT_AMPLITUDE, 0, VibrationEffect.DEFAULT_AMPLITUDE)
        vibrator.vibrate(VibrationEffect.createWaveform(pattern, amplitudes, -1))
    }
}

fun shouldAnnounceManeuverAdvance(previousIndex: Int, nextIndex: Int): Boolean {
    return nextIndex > previousIndex
}
