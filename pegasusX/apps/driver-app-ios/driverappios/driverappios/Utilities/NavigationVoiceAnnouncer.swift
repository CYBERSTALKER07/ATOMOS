//
//  NavigationVoiceAnnouncer.swift
//  driverappios
//

import AVFoundation

@MainActor
final class NavigationVoiceAnnouncer {
    private let synthesizer = AVSpeechSynthesizer()

    func announce(_ cue: NavigationCue) {
        let instruction = cue.instruction.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !instruction.isEmpty else { return }

        activateVoicePromptSession()
        Haptics.maneuver()
        synthesizer.stopSpeaking(at: .immediate)

        let utterance = AVSpeechUtterance(string: instruction)
        utterance.voice = AVSpeechSynthesisVoice(language: Locale.preferredLanguages.first ?? "en-US")
        utterance.rate = AVSpeechUtteranceDefaultSpeechRate
        synthesizer.speak(utterance)
    }

    func stop() {
        synthesizer.stopSpeaking(at: .immediate)
    }

    private func activateVoicePromptSession() {
        let session = AVAudioSession.sharedInstance()
        try? session.setCategory(.playback, mode: .voicePrompt, options: [.duckOthers])
        try? session.setActive(true)
    }
}

func shouldAnnounceManeuverAdvance(previousIndex: Int, nextIndex: Int) -> Bool {
    nextIndex > previousIndex
}
