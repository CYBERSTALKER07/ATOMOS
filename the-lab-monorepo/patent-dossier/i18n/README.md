# Pegasus Patent Dossier I18N Index

**Translation Posture**
1. Lead with operator benefit and system elegance, not component inventory.
1. Translate each packet as an architectural brief for Pegasus rather than as a dry software description.
1. Preserve actor-neutral language where possible so the filing remains future-proof across human and machine embodiments.

**Coverage**
1. Full localized core packet: Russian.
1. Full localized core packet: Uzbek.
1. Filing-ready native review pack: Russian covering a high-priority subset of patent-critical surfaces.
1. Filing-ready native review pack: Uzbek covering a high-priority subset of patent-critical surfaces.
1. Detailed page-dossier overlay: Russian covering all 59 page-dossier JSON files.
1. Detailed page-dossier overlay: Uzbek covering all 59 page-dossier JSON files.
1. Multilingual abstract support: English, Russian, Uzbek, Turkish, Spanish, French, German, Arabic, Simplified Chinese, Hindi.
1. Multilingual terminology support: English, Russian, Uzbek, Turkish, Spanish, French, German, Arabic, Simplified Chinese, Hindi.
1. Multilingual figure caption templates: English, Russian, Uzbek, Turkish, Spanish, French, German, Arabic, Simplified Chinese, Hindi.

**Files**
1. `patent-packet.ru.md`: Russian legal and strategic patent packet covering claim families, figure groups, backend automation architecture, and autonomy roadmap.
1. `patent-packet.uz.md`: Uzbek legal and strategic patent packet covering claim families, figure groups, backend automation architecture, and autonomy roadmap.
1. `filing-review-pack-high-priority.manifest.json`: machine-readable manifest fixing the exact high-priority surface subset used for the filing-ready native review packs.
1. `filing-review-pack-high-priority.ru.md`: Russian native-language review pack covering the selected high-priority surfaces across onboarding, supplier control, retailer commerce, driver execution, and payload dispatch.
1. `filing-review-pack-high-priority.uz.md`: Uzbek native-language review pack covering the selected high-priority surfaces across onboarding, supplier control, retailer commerce, driver execution, and payload dispatch.
1. `page-dossiers.ru.json`: Russian machine-readable overlay for every file in `patent-dossier/page-dossiers`, preserving English source anchors for routes, endpoints, page IDs, and exact UI labels.
1. `page-dossiers.uz.json`: Uzbek machine-readable overlay for every file in `patent-dossier/page-dossiers`, preserving English source anchors for routes, endpoints, page IDs, and exact UI labels.
1. `multilingual-abstracts.json`: short patent-grade abstract set for rapid review in ten languages.
1. `terminology-glossary.multilingual.json`: core vocabulary mapping for role names, checkout terms, routing, telemetry, offline proof, reverse logistics, and machine-native transition.
1. `figure-caption-templates.multilingual.json`: reusable caption text for figure production across common patent plates.

**Suggested Use**
1. Use the Russian and Uzbek markdown packets for counsel review, business review, and regional filing preparation.
1. Use the Russian and Uzbek high-priority review packs when a filing team needs native-language narrative prose rather than source-anchored overlay JSON.
1. Use the Russian and Uzbek page-dossier overlays when the filing or review process needs page-by-page UI, flow, button, icon, and state coverage without changing the original English source corpus.
1. Use the multilingual JSON files for fast translation alignment, figure generation, caption templating, and future expansion into additional full packets.
1. Extend this folder by adding one full markdown packet per language while keeping the JSON keys stable for tooling compatibility.

**Regeneration**
1. Rebuild the page-dossier overlays with: `node patent-dossier/tools/build-page-dossier-localizations.mjs`