package com.pegasusx.factory.data.remote

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class WebSocketTests {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun testRealtimeEventParsing() {
        val raw = """{"type":"FACTORY_MANIFEST_UPDATE"}"""
        val event = json.decodeFromString<FactoryLiveEvent>(raw)
        assertEquals(FactoryRealtimeEventType.ManifestUpdate, event.eventType)
    }

    @Test
    fun testRealtimeEventParsing_TransferUpdate() {
        val raw = """{"type":"FACTORY_TRANSFER_UPDATE"}"""
        val event = json.decodeFromString<FactoryLiveEvent>(raw)
        assertEquals(FactoryRealtimeEventType.TransferUpdate, event.eventType)
    }

    @Test
    fun testUnknownEventParsing() {
        val raw = """{"type":"FACTORY_UNKNOWN_EVENT"}"""
        val event = json.decodeFromString<FactoryLiveEvent>(raw)
        assertNull(event.eventType)
        assertEquals("FACTORY_UNKNOWN_EVENT", event.type)
    }
}
