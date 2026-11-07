# Original User Request

## Initial Request — 2026-09-26T17:09:44Z

# Teamwork Project Prompt — Draft

> Status: Ready for launch — awaiting user approval
> Goal: Craft prompt → get user approval → delegate to teamwork_preview
> Requested team: Full team

Analyze the real codebase for the Pegasus, PegasusX, and Pegasus.x ecosystems to generate accurate, enterprise-grade documentation and AI instructions. Create an `agents.md` and detailed feature/infra docs inside each respective app folder. Every technical claim must be grounded in reality ("no theatre").

Working directory: /Users/shakhzod/Desktop/V.O.I.D
Integrity mode: development

## Requirements

### R1. Ecosystem Instructions (`agents.md`)
Create a comprehensive `agents.md` file in the root of each ecosystem (`pegasus`, `pegasusX`, `pegasus.x`). These files must contain the final goal, honesty rules, and clear instructions tailored to that specific scale and architecture.

### R2. Feature and Infrastructure Documentation
Document the technical and non-technical implementation details of every feature, backend service, and infrastructure component for each of the three ecosystems. Explain what it is, how it works, and why it is there.

### R3. Absolute Code Grounding
Use semantic search, raw codebase reading, and RAG methods to achieve 100% accuracy. You must strictly avoid hallucination ("theatre") by basing every claim entirely on the actual source code.

## Acceptance Criteria

### Documentation Generation
- [ ] `pegasus/agents.md`, `pegasusX/agents.md`, and `pegasus.x/agents.md` exist and contain customized instructions.
- [ ] Detailed documentation is generated for the backend, features, and infrastructure of each ecosystem.

### Verification of Accuracy
- [ ] Every technical claim, architecture pattern, or feature mentioned in the generated documentation includes a direct file link (e.g., `file:///path/to/file`) to the exact source code proving its existence.
