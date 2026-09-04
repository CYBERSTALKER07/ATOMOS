import os
import glob
import re

files = glob.glob('apps/**/*WebSocket.kt', recursive=True) + glob.glob('apps/**/*RealtimeClient.kt', recursive=True) + glob.glob('apps/**/*Socket.kt', recursive=True)

old_closed = """            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                socket = null
            }"""

new_closed = """            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                socket = null
                if (intentionalClose.get()) return
                val attempt = reconnectAttempt.getAndIncrement()
                if (attempt >= MAX_RECONNECT_ATTEMPTS) return
                val delay = ReconnectBackoff.delayMs(attempt, BASE_DELAY_MS, MAX_DELAY_MS, -1)
                reconnectTask?.cancel(false)
                reconnectTask = reconnectExecutor.schedule({ connect() }, delay, TimeUnit.MILLISECONDS)
            }"""

for fpath in files:
    with open(fpath, 'r') as f:
        content = f.read()
    if old_closed in content:
        content = content.replace(old_closed, new_closed)
        with open(fpath, 'w') as f:
            f.write(content)
        print(f"Patched {fpath}")

