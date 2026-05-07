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
1. `patent-algorithm-atlas.ru.md`: Russian formula-level atlas of implemented algorithms across backend-go, ai-worker, app-side protocols, and infrastructure reliability controls.
1. `patent-claim-skeleton.ru.md`: Russian filing-oriented claim scaffold with independent and dependent claim templates mapped to verified implementation evidence.
1. `patent-line-art-prompts.ru.md`: Russian prompt library for strict black-and-white patent figure generation with unified line-art constraints.
1. `future-autonomous-vision.ru.md`: Russian no-human-loop future embodiment covering autonomous trucks, robotic payload/warehouse operation, and backend-first execution governance.
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
1. Use `patent-algorithm-atlas.ru.md` when drafting formula-centric or algorithm-centric claims and dependent claim trees.
1. Use `patent-claim-skeleton.ru.md` as the fast starting point for constructing independent and dependent claim sets before legal finalization.
1. Use `patent-line-art-prompts.ru.md` to keep all generated figures in legally clean monochrome line-art style.
1. Use `future-autonomous-vision.ru.md` when extending filings from human-assisted logistics to machine-native autonomous embodiments.
1. Use the Russian and Uzbek high-priority review packs when a filing team needs native-language narrative prose rather than source-anchored overlay JSON.
1. Use the Russian and Uzbek page-dossier overlays when the filing or review process needs page-by-page UI, flow, button, icon, and state coverage without changing the original English source corpus.
1. Use the multilingual JSON files for fast translation alignment, figure generation, caption templating, and future expansion into additional full packets.
1. Extend this folder by adding one full markdown packet per language while keeping the JSON keys stable for tooling compatibility.

**Regeneration**
1. Rebuild the page-dossier overlays with: `node patent-dossier/tools/build-page-dossier-localizations.mjs`