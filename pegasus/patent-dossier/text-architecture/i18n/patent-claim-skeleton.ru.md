# Technical Patent Architecture: Скелет Патентных Притязаний Pegasus (RU)

Source Document: i18n/patent-claim-skeleton.ru.md
Generated At: 2026-05-07T14:16:57.461Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- Это drafting scaffold для ускорения подготовки формулы изобретения. Текст предназначен для юридической доработки патентным поверенным и не является финальной редакцией притязаний.
- 1. Формульная база реализованной логики: patent-algorithm-atlas.ru.md.
- 1. Будущая автономная эмбодиментация: future-autonomous-vision.ru.md.

## System Architecture
- Implementation Anchor: apps/backend-go/dispatch/binpack.go
- Implementation Anchor: apps/ai-worker/optimizer/solver.go
- Implementation Anchor: apps/ai-worker/optimizer/clarke_wright.go
- Implementation Anchor: apps/ai-worker/optimizer/two_opt.go
- Implementation Anchor: apps/backend-go/proximity/engine.go
- Implementation Anchor: apps/backend-go/proximity/recommendation.go
- Implementation Anchor: apps/backend-go/factory/look_ahead.go
- Implementation Anchor: apps/backend-go/factory/supply_lanes.go
- Implementation Anchor: apps/backend-go/factory/network_optimizer.go
- Implementation Anchor: apps/backend-go/payment/refund.go
- Implementation Anchor: apps/backend-go/payment/gateway_client.go
- Implementation Anchor: apps/backend-go/idempotency/middleware.go
- Implementation Anchor: apps/admin-portal/lib/auth.ts
- Implementation Anchor: apps/admin-portal/lib/usePolling.ts
- Implementation Anchor: apps/backend-go/outbox/emit.go
- Implementation Anchor: apps/backend-go/outbox/relay.go

## Feature Set
1. 1. Правила Конструирования Притязаний
2. 2. Независимые Пункты (Шаблон)
3. Независимый Пункт 1 (Способ)
4. Независимый Пункт 2 (Система)
5. Независимый Пункт 3 (Машиночитаемый Носитель)
6. 3. Зависимые Пункты К Независимому Пункту 1 (Способ)
7. 4. Зависимые Пункты К Независимому Пункту 2 (Система)
8. 5. Зависимые Пункты К Независимому Пункту 3 (Носитель)
9. 6. Embodiment Variant: Автономный Режим
10. 7. Карта Доказательной Привязки (Для Внутренней Подготовки)
11. 8. Что Передавать В Юридическую Финализацию

## Algorithmic and Logical Flow
- No algorithm or workflow section detected.

## Mathematical Formulations
- 1. Способ по пункту 1, где порядок маршрута рассчитывают по savings-оценке вида $saving(i,j)=d(depot,i)+d(depot,j)-d(i,j)$.
- 1. Система по пункту 2, где целевой запас определяют как $max(safety, ceil(demand*1.15))$.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- No explicit constraint block detected.

## Claims-Oriented Technical Elements
1. Feature family coverage includes 1. Правила Конструирования Притязаний; 2. Независимые Пункты (Шаблон); Независимый Пункт 1 (Способ); Независимый Пункт 2 (Система); Независимый Пункт 3 (Машиночитаемый Носитель); 3. Зависимые Пункты К Независимому Пункту 1 (Способ).
2. Mathematical or scoring expressions are explicitly used for optimization or estimation.
