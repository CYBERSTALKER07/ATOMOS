# @pegasusx/desktop-cache

This package provides a unified SQLite local cache and offline command queue for Tauri desktop applications.

## Security Warning (SQLCipher)
As part of the Ecosystem Adaptability & Edge Case Handling design, any mobile or desktop database storing offline commands (`pending_commands`) **MUST** be encrypted at rest if it contains PII (e.g. driver delivery information, customer names).
Tauri's `plugin-sql` should be compiled with `sqlcipher` feature flag natively to meet this requirement.
