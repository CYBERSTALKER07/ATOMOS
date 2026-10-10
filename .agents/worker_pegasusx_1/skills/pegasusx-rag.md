---
name: pegasusx-rag
description: "Use this skill to semantically search the PegasusX codebase across backend, frontend, and infra using the local ChromaDB RAG server."
risk: safe
source: local
date_added: "2026-08-29"
---

# PegasusX Codebase RAG

## Overview
You are working on the PegasusX monorepo. Because the codebase is massive, you must NOT hallucinate or guess imports, Go structs, Spanner DDL schema, or React component props. 

Instead, you have access to a local Agentic RAG system that has chunked the entire monorepo into AST boundaries (functions, structs, classes) and embedded them into a ChromaDB vector database.

## When to Use This Skill
Use this skill **proactively** before writing code, modifying architecture, or answering questions about how PegasusX works.
- When you need to know how a specific payload is structured.
- When you need to find the definition of a Kafka consumer or event.
- When you need to see the Spanner schema for a specific table.
- When you need to see how a UI component in the web or mobile app is implemented.

## Instructions (How to Use)
The RAG system is exposed as a Model Context Protocol (MCP) server named `pegasusx-rag`.

To query the database, you must use your `call_mcp_tool` capability or eager tool:
- `mcp_pegasusx-rag_semantic_code_search`
