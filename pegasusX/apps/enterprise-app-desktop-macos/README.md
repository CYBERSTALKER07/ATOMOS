# Pegasus Executive Console (macOS Native Desktop)

A native macOS desktop application engineered in **Swift** and **SwiftUI** implementing the **Pegasus Swiss CAD Dark Design System**.

> **Operational Scope**: Dedicated console for the administrator to **review and answer incoming client requests** submitted via the Pegasus website. Clients submit requests on the website only; this native app serves as the review, response, and CRM workflow hub.

---

## Key Operator Capabilities
1. **Live Request Review**:
   - Inspect client contact info (Name, Corporate Email, Role, Company, Network Scope).
   - Review operational specifications and messages submitted through the website.
   - Filter by Status (`NEW`, `IN REVIEW`, `CONTACTED`, `QUALIFIED`, `CLOSED`) and Type (`Demo`, `Pilot`, `General`).
2. **Direct Answer Studio**:
   - Quick response templates (`DEMO INVITE`, `PILOT ONBOARDING`, `SPEC CLARIFICATION`).
   - One-click outbound email delivery via `/api/admin/reply` (Resend/SMTP).
   - System Mail (`mailto:`) fallback integration.
   - Automatically advances workflow status to `CONTACTED` upon answering.
3. **Workflow Management**:
   - Fast state machine transitions (`In Review`, `Contacted`, `Qualified`, `Closed`).
   - Record deletion for spam/duplicate cleanup.

---

## Swiss CAD Design Tokens
- **Pitch Black**: `#000000`
- **Obsidian Panels**: `#0D0D0E`, Card Surfaces `#141417`
- **Electric Cyan**: `#A9EBF9` (interactive accent & telemetry)
- **Volt Lime**: `#FBFF63` (new requests & action tags)
- **Emerald Green**: `#8DDC96` (qualified status & live sync indicator)
- **Laser Orange**: `#FE5934` (alerts & deletions)
- **Borders**: Sharp 0px CAD line strokes (`1px`).

---

## Build & Run

### Terminal
```bash
cd apps/enterprise-app-desktop-macos
swift run
```

### Xcode
```bash
xed apps/enterprise-app-desktop-macos
```
Select the **EnterpriseDesktopApp** scheme and press **Cmd+R**.
