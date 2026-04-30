**Generatedat:** 2026-04-06

**Pageid:** payload-auth-loading

**State:** authLoading

**Platform:** react-native-tablet

**Role:** PAYLOAD

# Sourcefiles

- apps/payload-terminal/App.tsx

**Shell:** payload-terminal-state-shell

**Status:** implemented

**Purpose:** Payload terminal session-restore state shown while SecureStore token and worker context are being recovered at app startup.

# Layoutzones

**Zoneid:** restore-center

**Position:** centered full-screen state

## Contents

- restoring session text

---


# Buttonplacements


# Iconplacements


# Interactiveflows

**Flowid:** session-restore

## Steps

- App locks screen to landscape
- App reads payloader token, name, and supplier ID from SecureStore
- App exits authLoading into login or authenticated state

---


# Statevariants

- restoring-session state

# Figureblueprints

- payload auth restore splash state

