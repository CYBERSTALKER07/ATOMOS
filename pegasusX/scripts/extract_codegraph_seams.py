#!/usr/bin/env python3
"""
extract_codegraph_seams.py — PegasusX Cross-Boundary Seam Extractor.

Extracts the 5 foundational cross-boundary relationships across the polyglot codebase:
1. Client API Call -> Backend HTTP Route (:ApiClientMethod)-[:CONSUMES_ROUTE]->(:RouteEndpoint)
   plus Route -> Service (:RouteEndpoint)-[:CALLS_SERVICE]->(:ServiceMethod)
2. Domain Service/Repo -> Spanner Table (:RepositoryMethod)-[:MUTATES_TABLE|:READS_TABLE]->(:SpannerTable)
3. Transactional Outbox -> Domain Event / Outbox (:ServiceMethod)-[:EMITS_OUTBOX]->(:OutboxEmitter)
4. Kafka Ingestion -> Downstream Consumer (:EventDefinition)-[:ROUTED_TO_TOPIC]->(:KafkaTopic)-[:CONSUMED_BY]->(:KafkaConsumer)
5. WebSocket Realtime Hub -> Role Rooms (:KafkaConsumer)-[:FANOUT_WS_ROOM]->(:WSHubRoom)-[:RECEIVED_BY]->(:ClientApp)

Can export raw Cypher statements or execute them directly against Memgraph/Neo4j over Bolt.
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import time
from typing import Any, Dict, List, Set, Tuple


def normalize_route_path(path: str) -> str:
    """Normalize Chi/Gin (:param or {param}), Retrofit ({param}), and Swift string-interpolated routes."""
    p = path.strip()
    # Replace Gin params :param with {param}
    p = re.sub(r':([a-zA-Z0-9_]+)', r'{\1}', p)
    # Replace wildcards *param with {param}
    p = re.sub(r'\*([a-zA-Z0-9_]+)', r'{\1}', p)
    # Remove query params if present
    p = p.split('?')[0]
    # Ensure leading slash
    if not p.startswith('/'):
        p = '/' + p
    return p


# ---------------------------------------------------------------------------
# Seam 1: Client API Call -> Backend HTTP Route
# ---------------------------------------------------------------------------
def extract_backend_routes(backend_dir: str) -> List[Dict[str, str]]:
    routes = []
    # Chi router methods: r.Get, r.Post, gr.With(...).Patch, etc.
    chi_route_pattern = re.compile(
        r'\.(Get|Post|Put|Patch|Delete|Head|Options|GET|POST|PUT|PATCH|DELETE)\s*\(\s*"([^"]+)"\s*,\s*([^,\)\n]+)',
        re.MULTILINE
    )

    for root, _, files in os.walk(backend_dir):
        if "test" in root or "vendor" in root:
            continue
        for f in files:
            if f.endswith(".go") and ("routes" in f or "routes" in root or "handler" in f):
                path = os.path.join(root, f)
                rel_path = os.path.relpath(path, backend_dir)
                pkg = os.path.basename(root)
                try:
                    with open(path, "r", encoding="utf-8", errors="ignore") as fh:
                        content = fh.read()
                    for match in chi_route_pattern.finditer(content):
                        method, raw_route, handler = match.groups()
                        norm_route = normalize_route_path(raw_route)
                        handler_clean = handler.strip().strip("&")
                        routes.append({
                            "id": f"route_{method.upper()}_{norm_route}",
                            "method": method.upper(),
                            "path": norm_route,
                            "raw_path": raw_route,
                            "handler": handler_clean,
                            "file": rel_path,
                            "package": pkg
                        })
                except Exception as e:
                    print(f"[WARN] Failed to parse Go route file {path}: {e}", file=sys.stderr)
    return routes


def extract_client_api_calls(root_dir: str) -> List[Dict[str, str]]:
    client_calls = []

    # 1. Android Kotlin Retrofit interfaces
    retrofit_re = re.compile(
        r'@(GET|POST|PUT|PATCH|DELETE)\s*\(\s*"([^"]+)"\s*\)\s*(?:@(?:Headers|FormUrlEncoded|Multipart)[^\n]*\n\s*)*(?:suspend\s+)?fun\s+([a-zA-Z0-9_]+)',
        re.MULTILINE
    )
    for dirpath, _, files in os.walk(os.path.join(root_dir, "apps")):
        if "-android" in dirpath:
            role = dirpath.split("-android")[0].split("/")[-1].replace("apps/", "")
            for f in files:
                if f.endswith(".kt"):
                    kpath = os.path.join(dirpath, f)
                    with open(kpath, "r", encoding="utf-8", errors="ignore") as fh:
                        content = fh.read()
                    for match in retrofit_re.finditer(content):
                        http_method, raw_endpoint, fn_name = match.groups()
                        norm_path = normalize_route_path(raw_endpoint)
                        client_calls.append({
                            "id": f"android_{role}_{fn_name}",
                            "role": role,
                            "platform": "android",
                            "method_name": fn_name,
                            "http_method": http_method.upper(),
                            "target_path": norm_path,
                            "file": os.path.relpath(kpath, root_dir)
                        })

    # 2. iOS Swift API calls
    swift_endpoint_re = re.compile(
        r'(?:case\s+([a-zA-Z0-9_]+)|func\s+([a-zA-Z0-9_]+))\s*(?:\([^)]*\))?[^{=]*[=:]\s*(?:Endpoint\(|path:\s*)?"([^"]*v1/[^"]+)"',
        re.MULTILINE
    )
    swift_call_re = re.compile(
        r'(?:apiClient|request|client)\.(get|post|put|patch|delete)\s*\(\s*"([^"]*v1/[^"]+)"',
        re.MULTILINE | re.IGNORECASE
    )
    for dirpath, _, files in os.walk(os.path.join(root_dir, "apps")):
        if "-ios" in dirpath:
            role = dirpath.split("-ios")[0].split("/")[-1].replace("apps/", "")
            for f in files:
                if f.endswith(".swift"):
                    spath = os.path.join(dirpath, f)
                    with open(spath, "r", encoding="utf-8", errors="ignore") as fh:
                        content = fh.read()
                    for match in swift_endpoint_re.finditer(content):
                        case_name, fn_name, endpoint = match.groups()
                        name = case_name or fn_name or "endpoint"
                        norm_path = normalize_route_path(endpoint)
                        client_calls.append({
                            "id": f"ios_{role}_{name}",
                            "role": role,
                            "platform": "ios",
                            "method_name": name,
                            "http_method": "ANY",
                            "target_path": norm_path,
                            "file": os.path.relpath(spath, root_dir)
                        })
                    for match in swift_call_re.finditer(content):
                        method, endpoint = match.groups()
                        norm_path = normalize_route_path(endpoint)
                        client_calls.append({
                            "id": f"ios_{role}_{os.path.splitext(f)[0]}_{method}",
                            "role": role,
                            "platform": "ios",
                            "method_name": f"{os.path.splitext(f)[0]}.{method}",
                            "http_method": method.upper(),
                            "target_path": norm_path,
                            "file": os.path.relpath(spath, root_dir)
                        })

    # 3. TypeScript / Web / Desktop api-core
    ts_core_path = os.path.join(root_dir, "packages/api-core/index.ts")
    if os.path.exists(ts_core_path):
        ts_call_re = re.compile(
            r'([a-zA-Z0-9_]+)\s*:\s*(?:async\s*)?\([^)]*\)\s*=>[^{]*{\s*(?:return\s+)?(?:this\.)?(?:client|request|get|post|put|patch|del)\.(get|post|put|patch|delete|request)\s*(?:<[^>]+>)?\s*\(\s*`?([^`",\)]+)`?',
            re.MULTILINE
        )
        with open(ts_core_path, "r", encoding="utf-8", errors="ignore") as fh:
            content = fh.read()
        for match in ts_call_re.finditer(content):
            fn_name, method, raw_path = match.groups()
            if "/v1/" in raw_path:
                norm_path = normalize_route_path(re.sub(r'\${[^}]+}', '{param}', raw_path))
                client_calls.append({
                    "id": f"ts_shared_{fn_name}",
                    "role": "shared",
                    "platform": "ts",
                    "method_name": fn_name,
                    "http_method": method.upper(),
                    "target_path": norm_path,
                    "file": "packages/api-core/index.ts"
                })

    return client_calls


# ---------------------------------------------------------------------------
# Seam 2: Service/Repo -> Spanner Entity Persistence
# ---------------------------------------------------------------------------
def extract_spanner_tables(ddl_path: str) -> List[Dict[str, Any]]:
    tables = []
    if not os.path.exists(ddl_path):
        return tables

    table_pattern = re.compile(
        r'CREATE TABLE\s+([A-Za-z0-9_]+)\s*\((.*?)\)\s*PRIMARY KEY',
        re.DOTALL | re.IGNORECASE
    )
    with open(ddl_path, "r", encoding="utf-8", errors="ignore") as fh:
        content = fh.read()

    for match in table_pattern.finditer(content):
        tname, cols_def = match.groups()
        has_supplier_id = "SupplierId" in cols_def
        tables.append({
            "name": tname,
            "has_supplier_id": has_supplier_id,
            "is_tenant_isolated": has_supplier_id
        })
    return tables


def extract_repo_spanner_bindings(backend_dir: str, spanner_tables: Set[str]) -> List[Dict[str, Any]]:
    bindings = []
    func_pattern = re.compile(r'func\s+\([^)]*\s*\*?([A-Za-z0-9_]+Repository)\)\s*([A-Za-z0-9_]+)\s*\([^)]*\)', re.MULTILINE)

    for root, _, files in os.walk(backend_dir):
        for f in files:
            if "repository_spanner" in f and f.endswith(".go") and not f.endswith("_test.go"):
                path = os.path.join(root, f)
                rel_path = os.path.relpath(path, backend_dir)
                with open(path, "r", encoding="utf-8", errors="ignore") as fh:
                    content = fh.read()

                funcs = content.split("func ")
                for chunk in funcs[1:]:
                    chunk_text = "func " + chunk
                    m = func_pattern.search(chunk_text)
                    if m:
                        repo_name, fn_name = m.groups()
                        for table in spanner_tables:
                            if re.search(rf'\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM|spanner\.Insert|spanner\.Update|spanner\.InsertOrUpdate)\b[^\n]*\b{table}\b', chunk_text, re.IGNORECASE) or \
                               re.search(rf'table:\s*"{table}"', chunk_text) or \
                               (table in chunk_text and any(k in chunk_text for k in ["txn.BufferWrite", "Apply(", "ReadWriteTransaction"])):
                                bindings.append({
                                    "repo": repo_name,
                                    "method": fn_name,
                                    "table": table,
                                    "action": "MUTATES",
                                    "file": rel_path
                                })
                            elif re.search(rf'\b(FROM|JOIN)\s+{table}\b', chunk_text, re.IGNORECASE) or \
                                 re.search(rf'client\.Single\(\)\.Read[^\n]*"{table}"', chunk_text):
                                bindings.append({
                                    "repo": repo_name,
                                    "method": fn_name,
                                    "table": table,
                                    "action": "READS",
                                    "file": rel_path
                                })
    return bindings


# ---------------------------------------------------------------------------
# Seam 3: Transactional Outbox -> Domain Event Emitter
# ---------------------------------------------------------------------------
def extract_outbox_emissions(backend_dir: str) -> List[Dict[str, str]]:
    emissions = []
    # Match: outbox.EmitJSON(ctx, txn, aggregateType, aggregateID, topic, payload)
    emit_pattern = re.compile(
        r'outbox\.EmitJSON\s*\([^,]+,\s*[^,]+,\s*(?:events\.)?([A-Za-z0-9_]+|"[^"]+"),\s*[^,]+,\s*(?:events\.)?([A-Za-z0-9_]+|"[^"]+")',
        re.MULTILINE
    )
    func_pattern = re.compile(r'func\s+\([^)]*\s*\*?([A-Za-z0-9_]+(?:Service|Repo|Handler)?)\)\s*([A-Za-z0-9_]+)', re.MULTILINE)

    for root, _, files in os.walk(backend_dir):
        for f in files:
            if f.endswith(".go") and not f.endswith("_test.go"):
                path = os.path.join(root, f)
                with open(path, "r", encoding="utf-8", errors="ignore") as fh:
                    content = fh.read()

                for chunk in content.split("func "):
                    if not chunk.strip():
                        continue
                    full_chunk = "func " + chunk
                    fm = func_pattern.search(full_chunk)
                    svc = fm.group(1) if fm else os.path.basename(root)
                    fn = fm.group(2) if fm else "handler"

                    for em in emit_pattern.finditer(full_chunk):
                        raw_agg = em.group(1).replace('"', '').replace('events.', '').replace('Aggregate', '')
                        raw_topic = em.group(2).replace('"', '').replace('events.', '')
                        # Normalize topic
                        topic_name = raw_topic.replace("Topic", "pegasusx-").lower()
                        if "main" in topic_name:
                            topic_name = "pegasusx-main"
                        elif "orders" in topic_name:
                            topic_name = "pegasusx-orders"
                        elif "dispatch" in topic_name:
                            topic_name = "pegasusx-dispatch"
                        elif "exceptions" in topic_name:
                            topic_name = "logistics.exceptions.v1"

                        emissions.append({
                            "service": svc,
                            "method": fn,
                            "aggregate": raw_agg,
                            "topic": topic_name,
                            "file": os.path.relpath(path, backend_dir)
                        })
    return emissions


# ---------------------------------------------------------------------------
# Seam 4: Kafka Ingestion -> Downstream Consumer Handler
# ---------------------------------------------------------------------------
def extract_kafka_topic_routing(backend_dir: str) -> List[Dict[str, str]]:
    routings = []
    routing_file = os.path.join(backend_dir, "events/topic_routing.go")
    if not os.path.exists(routing_file):
        return routings

    with open(routing_file, "r", encoding="utf-8", errors="ignore") as fh:
        content = fh.read()

    cases = re.findall(r'case\s+([^:]+):\s*return\s+([A-Za-z0-9_]+)', content, re.DOTALL)
    for event_list, topic_var in cases:
        topic_name = topic_var.replace("Topic", "pegasusx-").lower()
        if "main" in topic_name:
            topic_name = "pegasusx-main"
        elif "orders" in topic_name:
            topic_name = "pegasusx-orders"
        elif "dispatch" in topic_name:
            topic_name = "pegasusx-dispatch"
        elif "exceptions" in topic_name:
            topic_name = "logistics.exceptions.v1"
        elif "realtime" in topic_name:
            topic_name = "pegasusx-realtime"

        for ev in event_list.split(","):
            ev = ev.strip()
            if ev.startswith("Event"):
                routings.append({
                    "event_const": ev,
                    "event_name": ev.replace("Event", ""),
                    "topic": topic_name
                })
    return routings


def extract_kafka_consumers(backend_dir: str) -> List[Dict[str, str]]:
    consumers = [
        {"consumer": "order_consumer", "package": "order", "topic": "pegasusx-orders", "file": "order/consumer.go"},
        {"consumer": "warehouse_consumer", "package": "warehouse", "topic": "pegasusx-dispatch", "file": "warehouse/consumer.go"},
        {"consumer": "twin_consumer", "package": "twin", "topic": "pegasusx-orders", "file": "twin/consumer.go"},
        {"consumer": "twin_consumer_dispatch", "package": "twin", "topic": "pegasusx-dispatch", "file": "twin/consumer.go"},
        {"consumer": "twin_consumer_realtime", "package": "twin", "topic": "pegasusx-realtime", "file": "twin/consumer.go"},
        {"consumer": "twin_consumer_eta", "package": "twin", "topic": "route.eta.updated", "file": "twin/consumer.go"},
        {"consumer": "returns_consumer", "package": "returns", "topic": "logistics.exceptions.v1", "file": "returns/event_consumer.go"},
        {"consumer": "claims_bridge_consumer", "package": "claims", "topic": "pegasusx-main", "file": "claims/quarantine.go"},
        {"consumer": "billing_tier_consumer", "package": "kafka", "topic": "pegasusx-orders", "file": "kafka/billing_tier_worker.go"},
        {"consumer": "partner_webhook_consumer", "package": "partner", "topic": "pegasusx-orders", "file": "partner/consumer.go"},
        {"consumer": "partner_exceptions_consumer", "package": "partner", "topic": "logistics.exceptions.v1", "file": "partner/consumer.go"},
        {"consumer": "notification_dispatcher", "package": "kafka", "topic": "pegasusx-main", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_orders", "package": "kafka", "topic": "pegasusx-orders", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_dispatch", "package": "kafka", "topic": "pegasusx-dispatch", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_realtime", "package": "kafka", "topic": "pegasusx-realtime", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_exceptions", "package": "kafka", "topic": "logistics.exceptions.v1", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_telemetry", "package": "kafka", "topic": "logistics.telemetry.v1", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_demand", "package": "kafka", "topic": "pegasusx-demand", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_score", "package": "kafka", "topic": "driver.score.updated", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_capacity", "package": "kafka", "topic": "capacity.zone.updated", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "notification_dispatcher_demand_adj", "package": "kafka", "topic": "demand.adjustment.updated", "file": "kafka/notification_dispatcher.go"},
        {"consumer": "ai_freeze_lock_consumer", "package": "ai-worker", "topic": "pegasusx-freezelocks", "file": "apps/ai-worker/main.go"},
        {"consumer": "ai_inventory_import_consumer", "package": "ai-worker", "topic": "pegasusx-inventoryimportevents", "file": "apps/ai-worker/import_worker.go"},
    ]
    return consumers


# ---------------------------------------------------------------------------
# Seam 5: WebSocket Realtime Hub -> Role Rooms & Client Apps
# ---------------------------------------------------------------------------
def extract_websocket_hubs() -> List[Dict[str, str]]:
    roles = [
        {"role": "supplier", "hub": "SupplierHub", "app_ios": "apps/supplier-app-ios", "app_android": "apps/supplier-app-android", "app_web": "apps/supplier-portal"},
        {"role": "retailer", "hub": "RetailerHub", "app_ios": "apps/retailer-app-ios", "app_android": "apps/retailer-app-android", "app_web": "apps/retailer-app-desktop"},
        {"role": "driver", "hub": "DriverHub", "app_ios": "apps/driver-app-ios", "app_android": "apps/driver-app-android", "app_web": ""},
        {"role": "warehouse", "hub": "WarehouseHub", "app_ios": "apps/warehouse-app-ios", "app_android": "apps/warehouse-app-android", "app_web": "apps/warehouse-portal"},
        {"role": "factory", "hub": "FactoryHub", "app_ios": "apps/factory-app-ios", "app_android": "apps/factory-app-android", "app_web": "apps/factory-portal"},
        {"role": "payload", "hub": "PayloadHub", "app_ios": "apps/payload-app-ios", "app_android": "apps/payload-app-android", "app_web": "apps/payload-terminal"},
        {"role": "admin", "hub": "PlatformAdminHub", "app_ios": "", "app_android": "", "app_web": "apps/admin-portal"},
    ]
    return roles


# ---------------------------------------------------------------------------
# Cypher Statement Generator
# ---------------------------------------------------------------------------
def build_all_cypher(root_dir: str) -> List[str]:
    statements = [
        "// === Seam 1-5 Indexes & Constraints ===",
        "CREATE INDEX ON :RouteEndpoint(path);",
        "CREATE INDEX ON :RouteEndpoint(method);",
        "CREATE INDEX ON :ApiClientMethod(target_path);",
        "CREATE INDEX ON :ApiClientMethod(platform);",
        "CREATE INDEX ON :SpannerTable(name);",
        "CREATE INDEX ON :RepositoryMethod(repo);",
        "CREATE INDEX ON :EventDefinition(name);",
        "CREATE INDEX ON :KafkaTopic(name);",
        "CREATE INDEX ON :KafkaConsumer(consumer);",
        "CREATE INDEX ON :WSHubRoom(role);",
        "CREATE INDEX ON :OutboxEmitter(aggregate);",
    ]

    backend_dir = os.path.join(root_dir, "apps/backend-go")
    ddl_path = os.path.join(backend_dir, "schema/spanner.ddl")

    # 1. Backend Routes (Chi Router)
    routes = extract_backend_routes(backend_dir)
    for r in routes:
        statements.append(
            f"MERGE (re:RouteEndpoint {{id: '{r['id']}'}}) "
            f"SET re.method = '{r['method']}', re.path = '{r['path']}', "
            f"re.handler = '{r['handler']}', re.package = '{r['package']}', re.file = '{r['file']}';"
        )

    # 2. Client API Calls
    client_calls = extract_client_api_calls(root_dir)
    for c in client_calls:
        statements.append(
            f"MERGE (c:ApiClientMethod {{id: '{c['id']}'}}) "
            f"SET c.role = '{c['role']}', c.platform = '{c['platform']}', "
            f"c.method_name = '{c['method_name']}', c.target_path = '{c['target_path']}', "
            f"c.file = '{c['file']}';"
        )
        # Link to RouteEndpoint
        statements.append(
            f"MATCH (c:ApiClientMethod {{id: '{c['id']}'}}), (re:RouteEndpoint {{path: '{c['target_path']}'}}) "
            f"MERGE (c)-[:CONSUMES_ROUTE]->(re);"
        )

    # 3. Spanner Tables & Repo persistence
    tables = extract_spanner_tables(ddl_path)
    table_names = {t["name"] for t in tables}
    for t in tables:
        statements.append(
            f"MERGE (st:SpannerTable {{name: '{t['name']}'}}) "
            f"SET st.has_supplier_id = {str(t['has_supplier_id']).lower()}, "
            f"st.is_tenant_isolated = {str(t['is_tenant_isolated']).lower()};"
        )

    repo_bindings = extract_repo_spanner_bindings(backend_dir, table_names)
    for b in repo_bindings:
        rid = f"{b['repo']}_{b['method']}"
        statements.append(
            f"MERGE (rm:RepositoryMethod {{id: '{rid}'}}) "
            f"SET rm.repo = '{b['repo']}', rm.method = '{b['method']}', rm.file = '{b['file']}';"
        )
        rel = "MUTATES_TABLE" if b["action"] == "MUTATES" else "READS_TABLE"
        statements.append(
            f"MATCH (rm:RepositoryMethod {{id: '{rid}'}}), (st:SpannerTable {{name: '{b['table']}'}}) "
            f"MERGE (rm)-[:{rel}]->(st);"
        )

    # 4. Outbox Emissions
    emissions = extract_outbox_emissions(backend_dir)
    for e in emissions:
        svc_id = f"{e['service']}_{e['method']}"
        emitter_id = f"outbox_{e['aggregate']}_{e['topic']}"
        statements.append(
            f"MERGE (sm:ServiceMethod {{id: '{svc_id}'}}) "
            f"SET sm.service = '{e['service']}', sm.method = '{e['method']}', sm.file = '{e['file']}';"
        )
        statements.append(
            f"MERGE (oe:OutboxEmitter {{id: '{emitter_id}'}}) "
            f"SET oe.aggregate = '{e['aggregate']}', oe.topic = '{e['topic']}';"
        )
        statements.append(
            f"MATCH (sm:ServiceMethod {{id: '{svc_id}'}}), (oe:OutboxEmitter {{id: '{emitter_id}'}}) "
            f"MERGE (sm)-[:EMITS_OUTBOX]->(oe);"
        )
        statements.append(f"MERGE (kt:KafkaTopic {{name: '{e['topic']}'}});")
        statements.append(
            f"MATCH (oe:OutboxEmitter {{id: '{emitter_id}'}}), (kt:KafkaTopic {{name: '{e['topic']}'}}) "
            f"MERGE (oe)-[:ROUTED_TO_TOPIC]->(kt);"
        )

    # 5. Kafka Topics & Domain Event Mappings
    routings = extract_kafka_topic_routing(backend_dir)
    for rt in routings:
        statements.append(f"MERGE (ev:EventDefinition {{name: '{rt['event_name']}', const: '{rt['event_const']}'}});")
        statements.append(f"MERGE (kt:KafkaTopic {{name: '{rt['topic']}'}});")
        statements.append(
            f"MATCH (ev:EventDefinition {{name: '{rt['event_name']}'}}), (kt:KafkaTopic {{name: '{rt['topic']}'}}) "
            f"MERGE (ev)-[:ROUTED_TO_TOPIC]->(kt);"
        )

    consumers = extract_kafka_consumers(backend_dir)
    for kc in consumers:
        statements.append(
            f"MERGE (kc:KafkaConsumer {{consumer: '{kc['consumer']}'}}) "
            f"SET kc.package = '{kc['package']}', kc.file = '{kc['file']}';"
        )
        statements.append(
            f"MATCH (kt:KafkaTopic {{name: '{kc['topic']}'}}), (kc:KafkaConsumer {{consumer: '{kc['consumer']}'}}) "
            f"MERGE (kt)-[:CONSUMED_BY]->(kc);"
        )

    # 6. WebSocket Hub Rooms & Clients
    ws_roles = extract_websocket_hubs()
    for ws in ws_roles:
        rname = ws["role"]
        statements.append(
            f"MERGE (hub:WSHubRoom {{role: '{rname}', hub: '{ws['hub']}'}});"
        )
        statements.append(
            f"MATCH (kc:KafkaConsumer), (hub:WSHubRoom {{role: '{rname}'}}) "
            f"WHERE kc.package = '{rname}' OR kc.package = 'kafka' "
            f"MERGE (kc)-[:FANOUT_WS_ROOM]->(hub);"
        )
        if ws["app_ios"]:
            statements.append(f"MERGE (app:ClientApp {{id: '{rname}_ios', platform: 'ios', role: '{rname}', path: '{ws['app_ios']}'}});")
            statements.append(f"MATCH (hub:WSHubRoom {{role: '{rname}'}}), (app:ClientApp {{id: '{rname}_ios'}}) MERGE (hub)-[:RECEIVED_BY]->(app);")
        if ws["app_android"]:
            statements.append(f"MERGE (app:ClientApp {{id: '{rname}_android', platform: 'android', role: '{rname}', path: '{ws['app_android']}'}});")
            statements.append(f"MATCH (hub:WSHubRoom {{role: '{rname}'}}), (app:ClientApp {{id: '{rname}_android'}}) MERGE (hub)-[:RECEIVED_BY]->(app);")
        if ws["app_web"]:
            statements.append(f"MERGE (app:ClientApp {{id: '{rname}_web', platform: 'web', role: '{rname}', path: '{ws['app_web']}'}});")
            statements.append(f"MATCH (hub:WSHubRoom {{role: '{rname}'}}), (app:ClientApp {{id: '{rname}_web'}}) MERGE (hub)-[:RECEIVED_BY]->(app);")

    return statements


def execute_cypher_stream(bolt_url: str, user: str, password: str, statements: List[str]) -> None:
    try:
        from neo4j import GraphDatabase
    except ImportError:
        print("[ERROR] 'neo4j' Python package not found. Install via: pip install neo4j or uv pip install neo4j", file=sys.stderr)
        sys.exit(1)

    print(f"[*] Connecting to Memgraph/Neo4j at {bolt_url}...")
    auth = (user, password) if user or password else None
    driver = GraphDatabase.driver(bolt_url, auth=auth)

    success_count = 0
    with driver.session() as session:
        for stmt in statements:
            if stmt.startswith("//"):
                continue
            try:
                session.run(stmt)
                success_count += 1
            except Exception as e:
                if "already exists" in str(e).lower():
                    continue
                print(f"[WARN] Failed: {stmt[:60]}... -> {e}")

    driver.close()
    print(f"[SUCCESS] Executed {success_count} Cypher queries successfully.")


def main():
    parser = argparse.ArgumentParser(description="Extract PegasusX 5 Cross-Boundary Seams for CodeGraph")
    parser.add_argument(
        "--root-dir",
        default=os.path.abspath(os.path.join(os.path.dirname(__file__), "..")),
        help="Root path to pegasusX directory",
    )
    parser.add_argument("--bolt-url", default=os.getenv("MEMGRAPH_BOLT_URL", "bolt://localhost:7687"))
    parser.add_argument("--user", default=os.getenv("MEMGRAPH_USER", ""))
    parser.add_argument("--password", default=os.getenv("MEMGRAPH_PASSWORD", ""))
    parser.add_argument("--export-cypher", default="", help="Export Cypher script to file")
    parser.add_argument("--dry-run", action="store_true", help="Perform extraction without mutating database")

    args = parser.parse_args()
    start_time = time.time()
    print(f"[*] Starting Cross-Boundary Seam extraction on: {args.root_dir}")

    statements = build_all_cypher(args.root_dir)
    print(f"[+] Extracted {len(statements)} Cypher seam statements in {time.time() - start_time:.2f}s")

    if args.export_cypher:
        with open(args.export_cypher, "w", encoding="utf-8") as fh:
            fh.write("\n".join(statements) + "\n")
        print(f"[+] Exported seam statements to: {args.export_cypher}")

    if args.dry_run:
        print("[*] Dry run complete. No DB writes performed.")
        return

    if not args.export_cypher or args.bolt_url:
        execute_cypher_stream(args.bolt_url, args.user, args.password, statements)


if __name__ == "__main__":
    main()
