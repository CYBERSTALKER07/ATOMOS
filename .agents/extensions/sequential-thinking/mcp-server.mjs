#!/usr/bin/env node

import process from "node:process";

const SERVER_NAME = "sequential-thinking";
const SERVER_VERSION = "0.1.0";
const DEFAULT_PROTOCOL_VERSION = "2024-11-05";
const TRANSPORT_HEADERS = "headers";
const TRANSPORT_NDJSON = "ndjson";

const TOOL_DEFS = [
  {
    name: "sequential_thinking",
    description:
      "Break a problem into thought steps, including revisions and branches, and return structured thinking metadata.",
    inputSchema: {
      type: "object",
      required: ["thought", "nextThoughtNeeded", "thoughtNumber", "totalThoughts"],
      properties: {
        thought: {
          type: "string",
          description: "The current thinking step.",
        },
        nextThoughtNeeded: {
          type: "boolean",
          description: "Whether another thought is needed.",
        },
        thoughtNumber: {
          type: "integer",
          minimum: 1,
          description: "Current thought number in sequence.",
        },
        totalThoughts: {
          type: "integer",
          minimum: 1,
          description: "Estimated total thoughts needed.",
        },
        isRevision: {
          type: "boolean",
          description: "Whether this revises an earlier thought.",
        },
        revisesThought: {
          type: "integer",
          minimum: 1,
          description: "Thought number being revised.",
        },
        branchFromThought: {
          type: "integer",
          minimum: 1,
          description: "Thought number this branch started from.",
        },
        branchId: {
          type: "string",
          description: "Branch identifier when exploring alternatives.",
        },
        needsMoreThoughts: {
          type: "boolean",
          description: "Whether the estimate should be extended.",
        }
      },
      additionalProperties: false
    }
  }
];

function writeMessage(message) {
  const encoded = JSON.stringify(message);
  if (activeTransport === TRANSPORT_NDJSON) {
    process.stdout.write(`${encoded}\n`);
    return;
  }

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
      data
    }
  });
}

function asPositiveInt(value, field) {
  if (!Number.isInteger(value) || value < 1) {
    throw new Error(`${field} must be an integer greater than or equal to 1.`);
  }
  return value;
}

function asOptionalPositiveInt(value, field) {
  if (typeof value === "undefined") {
    return null;
  }
  return asPositiveInt(value, field);
}

function validateArguments(toolArgs) {
  if (!toolArgs || typeof toolArgs !== "object") {
    throw new Error("Tool arguments must be an object.");
  }

  if (typeof toolArgs.thought !== "string" || toolArgs.thought.trim().length === 0) {
    throw new Error("thought must be a non-empty string.");
  }

  if (typeof toolArgs.nextThoughtNeeded !== "boolean") {
    throw new Error("nextThoughtNeeded must be a boolean.");
  }

  const thoughtNumber = asPositiveInt(toolArgs.thoughtNumber, "thoughtNumber");
  const totalThoughts = Math.max(
    thoughtNumber,
    asPositiveInt(toolArgs.totalThoughts, "totalThoughts")
  );

  if (typeof toolArgs.isRevision !== "undefined" && typeof toolArgs.isRevision !== "boolean") {
    throw new Error("isRevision must be a boolean when provided.");
  }

  if (typeof toolArgs.needsMoreThoughts !== "undefined" && typeof toolArgs.needsMoreThoughts !== "boolean") {
    throw new Error("needsMoreThoughts must be a boolean when provided.");
  }

  if (typeof toolArgs.branchId !== "undefined" && typeof toolArgs.branchId !== "string") {
    throw new Error("branchId must be a string when provided.");
  }

  return {
    thoughtNumber,
    totalThoughts,
    revisesThought: asOptionalPositiveInt(toolArgs.revisesThought, "revisesThought"),
    branchFromThought: asOptionalPositiveInt(toolArgs.branchFromThought, "branchFromThought")
  };
}

function buildToolPayload(toolArgs) {
  const { thoughtNumber, totalThoughts, revisesThought, branchFromThought } =
    validateArguments(toolArgs);

  const progressRatio = Math.min(1, thoughtNumber / totalThoughts);
  const isRevision = toolArgs.isRevision === true;
  const needsMoreThoughts = toolArgs.needsMoreThoughts === true;
  const branchId = typeof toolArgs.branchId === "string" ? toolArgs.branchId : null;

  let guidance = "Continue with the next discriminating step.";
  if (toolArgs.nextThoughtNeeded === false) {
    guidance = "Reasoning can conclude. Verify the result before finalizing.";
  } else if (needsMoreThoughts || thoughtNumber >= totalThoughts) {
    guidance = "Increase totalThoughts or close with a validated conclusion.";
  } else if (isRevision) {
    guidance = "Resolve the revision locally, then continue from the corrected branch.";
  }

  return {
    thoughtNumber,
    totalThoughts,
    nextThoughtNeeded: toolArgs.nextThoughtNeeded,
    needsMoreThoughts,
    revision: {
      isRevision,
      revisesThought
    },
    branch: branchId
      ? {
          branchId,
          branchFromThought
        }
      : null,
    progress: {
      current: thoughtNumber,
      total: totalThoughts,
      ratio: Number(progressRatio.toFixed(2)),
      percent: Math.round(progressRatio * 100)
    },
    thoughtHistoryLength: thoughtNumber,
    guidance
  };
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
        tools: {}
      },
      serverInfo: {
        name: SERVER_NAME,
        version: SERVER_VERSION
      }
    });
    return;
  }

  if (method === "notifications/initialized") {
    return;
  }

  if (method === "tools/list") {
    writeResult(id, {
      tools: TOOL_DEFS
    });
    return;
  }

  if (method === "tools/call") {
    try {
      const toolName = params?.name;
      const toolArgs = params?.arguments || {};

      if (toolName !== "sequential_thinking") {
        writeError(id, -32602, `Unknown tool: ${toolName || "<missing>"}`);
        return;
      }

      writeResult(id, {
        content: [
          {
            type: "text",
            text: JSON.stringify(buildToolPayload(toolArgs), null, 2)
          }
        ]
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
                error: error?.message || String(error)
              },
              null,
              2
            )
          }
        ]
      });
      return;
    }
  }

  if (typeof id !== "undefined") {
    writeError(id, -32601, `Method not found: ${method}`);
  }
}

let inputBuffer = Buffer.alloc(0);
let activeTransport = null;

function findHeaderBoundary(buffer) {
  const crlfEnd = buffer.indexOf("\r\n\r\n");
  const lfEnd = buffer.indexOf("\n\n");

  if (crlfEnd === -1 && lfEnd === -1) {
    return null;
  }

  if (crlfEnd === -1) {
    return { headerEnd: lfEnd, separatorLength: 2 };
  }

  if (lfEnd === -1) {
    return { headerEnd: crlfEnd, separatorLength: 4 };
  }

  if (crlfEnd <= lfEnd) {
    return { headerEnd: crlfEnd, separatorLength: 4 };
  }

  return { headerEnd: lfEnd, separatorLength: 2 };
}

function consumeInputBuffer() {
  while (true) {
    if (activeTransport !== TRANSPORT_NDJSON) {
      const boundary = findHeaderBoundary(inputBuffer);
      if (boundary) {
        const { headerEnd, separatorLength } = boundary;

        const headerRaw = inputBuffer.slice(0, headerEnd).toString("utf8");
        const lengthMatch = headerRaw.match(/content-length:\s*(\d+)/i);
        if (!lengthMatch) {
          inputBuffer = inputBuffer.slice(headerEnd + separatorLength);
          continue;
        }

        const length = Number.parseInt(lengthMatch[1], 10);
        if (Number.isNaN(length) || length < 0) {
          inputBuffer = inputBuffer.slice(headerEnd + separatorLength);
          continue;
        }

        const totalSize = headerEnd + separatorLength + length;
        if (inputBuffer.length < totalSize) {
          return;
        }

        const payload = inputBuffer
          .slice(headerEnd + separatorLength, totalSize)
          .toString("utf8");

        inputBuffer = inputBuffer.slice(totalSize);
        activeTransport = TRANSPORT_HEADERS;

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
              error?.message || String(error)
            );
          }
        });
        continue;
      }
    }

    if (activeTransport !== TRANSPORT_HEADERS) {
      const newlineIndex = inputBuffer.indexOf("\n");
      if (newlineIndex !== -1) {
        const rawLine = inputBuffer.slice(0, newlineIndex).toString("utf8");
        inputBuffer = inputBuffer.slice(newlineIndex + 1);

        const line = rawLine.replace(/\r$/, "").trim();
        if (!line) {
          continue;
        }

        if (!line.startsWith("{") && !line.startsWith("[")) {
          if (activeTransport === TRANSPORT_NDJSON) {
            continue;
          }
          return;
        }

        let message;
        try {
          message = JSON.parse(line);
        } catch {
          if (activeTransport === TRANSPORT_NDJSON) {
            continue;
          }
          return;
        }

        activeTransport = TRANSPORT_NDJSON;
        handleRpcMessage(message).catch((error) => {
          if (typeof message?.id !== "undefined") {
            writeError(
              message.id,
              -32000,
              "Internal server error",
              error?.message || String(error)
            );
          }
        });
        continue;
      }
    }

    return;
  }
}

process.stdin.on("data", (chunk) => {
  inputBuffer = Buffer.concat([inputBuffer, chunk]);
  consumeInputBuffer();
});

process.stdin.on("error", () => {
  process.exit(1);
});