import okhttp3.*

class test_onclosed : WebSocketListener() {
    override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
        // test
    }
}
