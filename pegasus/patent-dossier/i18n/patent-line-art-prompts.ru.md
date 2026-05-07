# Библиотека Промптов Для Патентных Иллюстраций (RU, Black-White Line Art)

**Назначение**
Документ задает единый production-стандарт для генерации патентных иллюстраций Pegasus в монохромной линейной манере: только черные линии на белом фоне, без декоративной графики.

## 1. Глобальный Визуальный Стандарт

1. Режим: black-and-white line art only.
1. Фон: чисто белый (#FFFFFF).
1. Линии: чисто черный (#000000), без серого и без полутонов.
1. Толщина линий:
   - контур модулей: 2.0 pt
   - внутренние связи: 1.2 pt
   - вторичные стрелки: 0.8 pt
1. Запрещено: градиенты, тени, текстуры, шум, фотографичность, 3D-рендер, цветовые акценты.
1. Тип проекции: ортографическая схема или строгая псевдоизометрия без перспективной дисторсии.
1. Подписи: uppercase, короткие технические лейблы, одинаковый шрифт sans-serif.
1. Стрелки: однотипные, односторонние, с ясным направлением потока.
1. Выходные форматы: SVG (primary), PDF (legal), PNG 4K (preview).

## 2. Универсальный Шаблон Промпта

Использовать как базу для любой фигуры:

"Patent technical diagram, strict black ink on white background, vector line art, no grayscale, no gradients, no shadows, no textures, clear module boxes, directional arrows, concise uppercase labels, high information density, publication-ready legal illustration"

Negative prompt (добавлять всегда):

"no color, no gray fill, no blur, no glow, no 3d render, no photorealism, no background pattern, no decorative icons, no perspective distortion"

## 3. Каталог Фигур И Промптов

### Фигура A. Multi-Role Operating Fabric
1. Сцена: SUPPLIER, RETAILER, DRIVER, PAYLOAD, FACTORY_ADMIN, WAREHOUSE_ADMIN как отдельные операционные зоны, сходящиеся в backend orchestration core.
1. Обязательные элементы: role boxes, auth boundary, event spine, data plane, ws channels.
1. Промпт: "Black-white line-art ecosystem map of six logistics roles connected to one orchestration backend, explicit auth boundaries, kafka event spine, spanner data core, websocket channels, orthographic technical patent plate"

### Фигура B. Dispatch Formula Pipeline
1. Сцена: dispatchable order intake -> H3 clustering -> binpack with 0.95 buffer -> route optimization (savings + 2-opt) -> manifest commit.
1. Обязательные элементы: equation callouts for saving(i,j), max_stops=25, two_opt_iter=200.
1. Промпт: "Monochrome patent flowchart of dispatch optimizer pipeline with formula annotations: saving(i,j)=d(depot,i)+d(depot,j)-d(i,j), tetris buffer 0.95, two-opt 200 iterations, max 25 stops"

### Фигура C. Geofence And Proximity Control
1. Сцена: driver coordinate stream, H3 cell index, proximity score thresholds, completion geofence gate.
1. Обязательные элементы: trigger>0.9, target<0.7, non-linear load penalty block.
1. Промпт: "Strict line-art geospatial control diagram showing H3 grid assignment, proximity reroute thresholds trigger above 0.9 and stabilization below 0.7, nonlinear warehouse load penalty gate"

### Фигура D. Replenishment Intelligence Loop
1. Сцена: pull-matrix breach detection + look-ahead forecast + predictive push + lock arbitration.
1. Обязательные элементы: look-ahead 7 days, safety buffer 15%, lock TTL 10m, EMA alpha 0.2.
1. Промпт: "Black-white technical loop diagram for replenishment: threshold breach scan, look-ahead window 7d, target stock max(safety, ceil(demand*1.15)), lane EMA alpha 0.2, lock TTL 10m"

### Фигура E. Payment Settlement Spine
1. Сцена: checkout session, webhook validation, idempotency guard, ledger write, supplier/driver notifications.
1. Обязательные элементы: exponential backoff formula, replay-safe keying, settlement status transition.
1. Промпт: "Patent payment orchestration line-art plate with hosted checkout session, signature-first webhook gate, idempotency replay shield, exponential retry delay 500*2^(attempt-1), ledger reconciliation"

### Фигура F. Offline Proof And Desert Sync
1. Сцена: mobile offline capture -> local queue -> sync retry -> redis dedup -> canonical commit.
1. Обязательные элементы: 24h stale drop gate, single-write confirmation, conflict path.
1. Промпт: "Monochrome legal-grade diagram of offline delivery proof pipeline: local capture, queued mutation replay, stale item drop after 24h, redis dedup key, single canonical commit"

### Фигура G. Outbox Causality Chain
1. Сцена: Spanner RW transaction writing domain row + outbox row -> relay -> kafka -> websocket invalidate.
1. Обязательные элементы: atomic boundary box around transaction.
1. Промпт: "Black ink patent causality diagram showing transactional outbox atomic region, relay publish, keyed kafka partitioning, websocket invalidation fanout"

### Фигура H. Backpressure And Priority Shed
1. Сцена: incoming traffic classes P0/P1/P2, overload detector, shed policy, client backpressure interval signal.
1. Обязательные элементы: rate limiter bucket, circuit breaker states.
1. Промпт: "Strict black-white architecture chart of overload defense: priority tiers, token bucket limiter, circuit breaker closed-open-half-open, backpressure interval feedback to clients"

### Фигура I. Driver Telemetry Filter
1. Сцена: raw GPS stream -> deviation filter -> websocket publish -> control tower update.
1. Обязательные элементы: distance>20m or bearing>15deg emission gate.
1. Промпт: "Monochrome signal-processing plate for driver telemetry with threshold gate distance over 20 meters or bearing over 15 degrees before publish"

### Фигура J. Autonomous 2051 Execution Mesh
1. Сцена: autonomous trucks, robotic loading cells, machine policy core, human removed from steady-state loop.
1. Обязательные элементы: exception-only human governance node separated from main loop.
1. Промпт: "Black-white future patent plate of fully autonomous logistics mesh: robotic warehouse, autonomous truck fleet, backend policy brain, exception-only governance side channel"

## 4. Редакционный Чеклист Перед Экспортом

1. Проверить, что в кадре нет ни одного цветного пикселя.
1. Проверить, что стрелки читаются в направлении причинности.
1. Проверить, что каждое уравнение читается без пояснительного абзаца.
1. Проверить, что фигура не зависит от конкретного бренда UI-компонента.
1. Проверить, что подписи короткие и машинно-нейтральные.

## 5. Связь С Остальными Материалами

1. Формульные источники: patent-algorithm-atlas.ru.md.
1. Стратегическая рамка и claim families: patent-packet.ru.md.
1. Автономный no-human-loop сценарий: future-autonomous-vision.ru.md.
