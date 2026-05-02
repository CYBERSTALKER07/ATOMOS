#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

const SERVER_NAME = "void-ast-engine";
const SERVER_VERSION = "0.1.0";
const DEFAULT_PROTOCOL_VERSION = "2024-11-05";

function workspaceRoot() {
  return process.cwd();
}

function defaultRoot() {
  const candidate = path.join(workspaceRoot(), "pegasus");
  if (fs.existsSync(candidate)) {
    return candidate;
  }
  return workspaceRoot();
}

const ENGINE_PATH =
  process.env.VOID_AST_ENGINE_PATH ||
  path.join(workspaceRoot(), ".agents", "extensions", "ast-engine", "engine.mjs");

const DEFAULT_ROOT = process.env.VOID_AST_ROOT || defaultRoot();

const DEFAULT_INDEX =
  process.env.VOID_AST_INDEX ||
  path.join(
    workspaceRoot(),
    ".agents",
    "extensions",
    "ast-engine",
    ".cache",
    "pegasus-index.json",
  );

const TOOL_DEFS = [
  {
    name: "void_ast_index",
    description:
      "Build or refresh the AST symbol index for the workspace before technical execution.",
    inputSchema: {
      type: "object",
      properties: {
        root: {
          type: "string",
          description: "Absolute or workspace-relative root path to scan. Defaults to pegasus/.",
        },
        index: {
          type: "string",
          description: "Absolute or workspace-relative path to the index JSON file.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "void_ast_definition",
    description:
      "Find symbol definitions and exact file/line locations for a target name.",
    inputSchema: {
      type: "object",
      required: ["symbol"],
      properties: {
        symbol: {
          type: "string",
          description: "Target symbol name, such as RegisterRoutes or NewApp.",
        },
        root: {
          type: "string",
          description: "Absolute or workspace-relative root path to scan. Defaults to pegasus/.",
        },
        index: {
          type: "string",
          description: "Absolute or workspace-relative path to the index JSON file.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "void_ast_usages",
    description:
      "Find usage references for a symbol to compute blast radius before edits.",
    inputSchema: {
      type: "object",
      required: ["symbol"],
      properties: {
        symbol: {
          type: "string",
          description: "Target symbol name, such as RegisterRoutes or NewApp.",
        },
        limit: {
          type: "number",
          description: "Maximum number of usage hits to return.",
        },
        root: {
          type: "string",
          description: "Absolute or workspace-relative root path to scan. Defaults to pegasus/.",
        },
        index: {
          type: "string",
          description: "Absolute or workspace-relative path to the index JSON file.",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "void_ast_graph",
    description:
      "Return definition, usage, impacted files, and import-neighbor graph for a symbol.",
    inputSchema: {
      type: "object",
      required: ["symbol"],
      properties: {
        symbol: {
          type: "string",
          description: "Target symbol name, such as RegisterRoutes or NewApp.",
        },
        limit: {
          type: "number",
          description: "Maximum number of usage hits to include.",
        },
        root: {
          type: "string",
          description: "Absolute or workspace-relative root path to scan. Defaults to pegasus/.",
        },
        index: {
          type: "string",
          description: "Absolute or workspace-relative path to the index JSON file.",
        },
      },
      additionalProperties: false,
    },
  },
];

function writeMessage(message) {
  const encoded = JSON.stringify(message);
  const contentLength = Buffer.byteLength(encoded, "utf8");
  process.stdout.write(`Content-Length: ${contentLength}\r\n\r\n${encoded}`);
}

function writeResult(id, result) {
  writeMessage({ jsonrpc: "2.0", id, result });
}

function writeError(id, code, message, data) {
  writeMessage({
    jsonrpc: "2.0",
    id,
    error: {
      code,
      message,
      data,
    },
  });
}

function safePath(input, fallback) {
  if (!input || typeof input !== "string") {
    return fallback;
  }

  if (path.isAbsolute(input)) {
    return input;
  }

  return path.resolve(workspaceRoot(), input);
}

async function runEngine(command, toolArgs) {
  const args = [
    ENGINE_PATH,
    command,
    "--root",
    safePath(toolArgs?.root, DEFAULT_ROOT),
    "--index",
    safePath(toolArgs?.index, DEFAULT_INDEX),
  ];

  if (toolArgs?.symbol) {
    args.push("--symbol", String(toolArgs.symbol));
  }

  if (typeof toolArgs?.limit === "number") {
    args.push("--limit", String(toolArgs.limit));
  }

  const { stdout, stderr } = await execFileAsync("node", args, {
    cwd: workspaceRoot(),
    maxBuffer: 10 * 1024 * 1024,
  });

  if (stderr && stderr.trim().length > 0) {
    throw new Error(stderr.trim());
  }

  const trimmed = stdout.trim();
  if (!trimmed) {
    throw new Error("AST engine returned empty output.");
  }

  return JSON.parse(trimmed);
}

async function callTool(name, toolArgs) {
  if (name === "void_ast_index") {
    return runEngine("index", toolArgs || {});
  }

  if (name === "void_ast_definition") {
    return runEngine("definition", toolArgs || {});
  }

  if (name === "void_ast_usages") {
    return runEngine("usages", toolArgs || {});
  }

  if (name === "void_ast_graph") {
    return runEngine("graph", toolArgs || {});
  }

  throw new Error(`Unknown tool: ${name}`);
}

async function handleRpcMessage(message) {
  if (!message || typeof message !== "object") {
    return;
  }

  const { id, method, params } = message;

  if (!method) {
    return;
  }

  if (method === "initialize") {
    const requestedVersion =
      params && typeof params === "object" && typeof params.protocolVersion === "string"
        ? params.protocolVersion
        : DEFAULT_PROTOCOL_VERSION;

    writeResult(id, {
      protocolVersion: requestedVersion,
      capabilities: {
        tools: {},
      },
      serverInfo: {
        name: SERVER_NAME,
        version: SERVER_VERSION,
      },
    });
    return;
  }

  if (method === "notifications/initialized") {
    return;
  }

  if (method === "tools/list") {
    writeResult(id, {
      tools: TOOL_DEFS,
    });
    return;
  }

  if (method === "tools/call") {
    try {
      const toolName = params?.name;
      const toolArgs = params?.arguments || {};

      if (!toolName || typeof toolName !== "string") {
        writeError(id, -32602, "Invalid params: tools/call requires tool name.");
        return;
      }

      const payload = await callTool(toolName, toolArgs);
      writeResult(id, {
        content: [
          {
            type: "text",
            text: JSON.stringify(payload, null, 2),
          },
        ],
      });
      return;
    } catch (error) {
      writeResult(id, {
        isError: true,
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                ok: false,
                error: error?.message || String(error),
              },
              null,
              2,
            ),
          },
        ],
      });
      return;
    }
  }

  if (typeof id !== "undefined") {
    writeError(id, -32601, `Method not found: ${method}`);
  }
}

let inputBuffer = Buffer.alloc(0);

function consumeInputBuffer() {
  while (true) {
    const headerEnd = inputBuffer.indexOf("\r\n\r\n");
    if (headerEnd === -1) {
      return;
    }

    const headerRaw = inputBuffer.slice(0, headerEnd).toString("utf8");
    const lengthMatch = headerRaw.match(/content-length:\s*(\d+)/i);
    if (!lengthMatch) {
      inputBuffer = inputBuffer.slice(headerEnd + 4);
      continue;
    }

    const length = Number.parseInt(lengthMatch[1], 10);
    if (Number.isNaN(length) || length < 0) {
      inputBuffer = inputBuffer.slice(headerEnd + 4);
      continue;
    }

    const totalSize = headerEnd + 4 + length;
    if (inputBuffer.length < totalSize) {
      return;
    }

    const payload = inputBuffer
      .slice(headerEnd + 4, totalSize)
      .toString("utf8");

    inputBuffer = inputBuffer.slice(totalSize);

    let message;
    try {
      message = JSON.parse(payload);
    } catch {
      continue;
    }

    handleRpcMessage(message).catch((error) => {
      if (typeof message?.id !== "undefined") {
        writeError(
          message.id,
          -32000,
          "Internal server error",
          error?.message || String(error),
        );
      }
    });
  }
}

process.stdin.on("data", (chunk) => {
  inputBuffer = Buffer.concat([inputBuffer, chunk]);
  consumeInputBuffer();
});

process.stdin.on("error", () => {
  process.exit(1);
});
