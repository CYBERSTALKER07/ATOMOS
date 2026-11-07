import { Bot, InlineKeyboard } from "grammy";
import * as dotenv from "dotenv";
import express, { Request, Response } from "express";
import cors from "cors";
import crypto from "crypto";

dotenv.config();

const token = process.env.TELEGRAM_BOT_TOKEN || process.env.BOT_TOKEN || "";
const backendBaseUrl = (process.env.PEGASUSX_BACKEND_URL || "http://localhost:8080").replace(/\/+$/, "");
const webhookSecret = process.env.BOT_WEBHOOK_SECRET || "";

const bot = token ? new Bot(token) : null;

interface LinkedSession {
  jwt: string;
  retailerId?: string;
  linkedAt: string;
}

const chatSessions = new Map<number, LinkedSession>();

export function deterministicKey(prefix: string, ...parts: string[]): string {
  const digest = crypto.createHash("sha256").update(parts.join(":")).digest("hex").slice(0, 32);
  return `${prefix}:${digest}`;
}

async function backendRequest<T>(
  path: string,
  method: "GET" | "POST",
  jwt: string,
  body?: unknown,
  idempotencyKey?: string,
): Promise<{ ok: boolean; status: number; data?: T; error?: string }> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    Authorization: `Bearer ${jwt}`,
  };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (idempotencyKey) {
    headers["Idempotency-Key"] = idempotencyKey;
  }

  try {
    const res = await fetch(`${backendBaseUrl}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: AbortSignal.timeout(10_000),
    });
    const text = await res.text();
    let parsed: unknown;
    try {
      parsed = text ? JSON.parse(text) : undefined;
    } catch {
      parsed = undefined;
    }
    if (!res.ok) {
      const errObj = parsed as { error?: string; detail?: string; message?: string } | undefined;
      return {
        ok: false,
        status: res.status,
        error: errObj?.detail || errObj?.error || errObj?.message || `HTTP ${res.status}`,
      };
    }
    return { ok: true, status: res.status, data: parsed as T };
  } catch (err) {
    return {
      ok: false,
      status: 503,
      error: err instanceof Error ? err.message : "backend unreachable",
    };
  }
}

if (bot) {
  bot.command("start", async (ctx) => {
    await ctx.reply(
      "Welcome to *PegasusX B2B Retailer Restock Assistant*!\n\n" +
        "Commands:\n" +
        "• `/link <retailer_jwt>` — Bind this chat to your Retailer account\n" +
        "• `/orders` — View active orders & preorders\n" +
        "• `/order <order_id>` — Confirm or reject a pending preorder\n" +
        "• `/status` — Check PegasusX backend connectivity",
      { parse_mode: "Markdown" },
    );
  });

  bot.command("link", async (ctx) => {
    const chatId = ctx.chat?.id;
    const jwt = String(ctx.match || "").trim();
    if (!chatId || !jwt) {
      await ctx.reply("Usage: `/link <retailer_jwt>`", { parse_mode: "Markdown" });
      return;
    }
    const check = await backendRequest<{ retailer_id?: string; id?: string }>(
      "/v1/retailer/profile",
      "GET",
      jwt,
    );
    if (!check.ok) {
      await ctx.reply(`❌ Could not verify retailer session: ${check.error}`);
      return;
    }
    const retailerId = check.data?.retailer_id || check.data?.id || "verified";
    chatSessions.set(chatId, {
      jwt,
      retailerId,
      linkedAt: new Date().toISOString(),
    });
    await ctx.reply(`✅ Linked Telegram chat to Retailer \`${retailerId}\`.`, {
      parse_mode: "Markdown",
    });
  });

  bot.command("orders", async (ctx) => {
    const chatId = ctx.chat?.id;
    const session = chatId ? chatSessions.get(chatId) : undefined;
    if (!session) {
      await ctx.reply("⚠️ Link your retailer session first with `/link <retailer_jwt>`.", {
        parse_mode: "Markdown",
      });
      return;
    }
    const res = await backendRequest<{
      orders?: Array<{ order_id: string; status: string; total_amount_minor?: number; currency?: string }>;
    }>("/v1/retailer/orders?limit=5", "GET", session.jwt);
    if (!res.ok) {
      await ctx.reply(`❌ Failed to load orders: ${res.error}`);
      return;
    }
    const orders = res.data?.orders ?? [];
    if (orders.length === 0) {
      await ctx.reply("No active orders found.");
      return;
    }
    const lines = orders.map(
      (o) =>
        `• \`${o.order_id}\` — *${o.status}*` +
        (typeof o.total_amount_minor === "number"
          ? ` (${o.total_amount_minor} ${o.currency || "UZS"} minor)`
          : ""),
    );
    await ctx.reply(`*Recent Orders:*\n${lines.join("\n")}`, { parse_mode: "Markdown" });
  });

  bot.on("callback_query:data", async (ctx) => {
    const data = ctx.callbackQuery.data;
    const chatId = ctx.chat?.id;
    const session = chatId ? chatSessions.get(chatId) : undefined;
    if (!session) {
      await ctx.answerCallbackQuery({
        text: "Link your retailer account first via /link <jwt>.",
        show_alert: true,
      });
      return;
    }

    if (data.startsWith("order_") || data.startsWith("confirm_")) {
      const orderId = data.replace(/^(order_|confirm_)/, "").trim();
      const idemKey = `retailer-confirm-preorder:${orderId}`;
      const result = await backendRequest(
        `/v1/order/${encodeURIComponent(orderId)}/confirm-preorder`,
        "POST",
        session.jwt,
        {},
        idemKey,
      );
      if (!result.ok) {
        await ctx.answerCallbackQuery({ text: `Confirmation failed: ${result.error}`, show_alert: true });
        return;
      }
      await ctx.answerCallbackQuery({ text: "Preorder confirmed in PegasusX!" });
      await ctx.editMessageText(
        `✅ *Preorder Confirmed*\nOrder ID: \`${orderId}\`\nIdempotency-Key: \`${idemKey}\``,
        { parse_mode: "Markdown" },
      );
    } else if (data.startsWith("cancel_")) {
      const orderId = data.replace("cancel_", "").trim();
      const reason = "rejected_via_telegram_bot";
      const idemKey = deterministicKey(`retailer-reject-preorder:${orderId}`, reason);
      const result = await backendRequest(
        `/v1/order/${encodeURIComponent(orderId)}/reject-preorder`,
        "POST",
        session.jwt,
        { reason },
        idemKey,
      );
      if (!result.ok) {
        await ctx.answerCallbackQuery({ text: `Rejection failed: ${result.error}`, show_alert: true });
        return;
      }
      await ctx.answerCallbackQuery({ text: "Preorder rejected." });
      await ctx.editMessageText(`❌ *Preorder Rejected*\nOrder ID: \`${orderId}\``, {
        parse_mode: "Markdown",
      });
    }
  });
}

export const app = express();
app.use(cors());
app.use(express.json({ limit: "64kb" }));

app.get("/healthz", (_req: Request, res: Response) => {
  res.status(200).json({
    status: "ok",
    service: "bot-telegram",
    bot_configured: Boolean(bot),
    backend_url: backendBaseUrl,
  });
});

// REST API for Go Backend / AI Worker to Trigger Proactive Restock Messages for a real Order ID
app.post("/trigger-restock", async (req: Request, res: Response) => {
  if (webhookSecret) {
    const provided = req.header("X-Bot-Webhook-Secret") || "";
    if (
      provided.length !== webhookSecret.length ||
      !crypto.timingSafeEqual(Buffer.from(provided), Buffer.from(webhookSecret))
    ) {
      res.status(401).json({ error: "unauthorized" });
      return;
    }
  }
  if (!bot) {
    res.status(503).json({ error: "telegram_bot_token_not_configured" });
    return;
  }

  try {
    const { telegram_chat_id, order_id, retailer_name, product_name, suggested_qty } = req.body as {
      telegram_chat_id?: number | string;
      order_id?: string;
      retailer_name?: string;
      product_name?: string;
      suggested_qty?: number;
    };

    if (!telegram_chat_id || !order_id) {
      res.status(400).json({ error: "telegram_chat_id and order_id are required" });
      return;
    }

    const message =
      `Hello ${retailer_name || "Retailer"} 👋\n\n` +
      `AI Replenishment Preorder \`${order_id}\` is ready` +
      (product_name ? ` for *${product_name}* (${suggested_qty ?? 0} units)` : "") +
      `.\n\nConfirm to release to the supplier warehouse:`;

    const keyboard = new InlineKeyboard()
      .text("✅ Confirm Preorder", `confirm_${order_id}`)
      .text("❌ Reject", `cancel_${order_id}`);

    await bot.api.sendMessage(telegram_chat_id, message, {
      parse_mode: "Markdown",
      reply_markup: keyboard,
    });

    res.json({ success: true, order_id, message: "Proactive restock preorder sent." });
  } catch (error: unknown) {
    const msg = error instanceof Error ? error.message : "telegram send failed";
    res.status(502).json({ error: msg });
  }
});

const PORT = Number(process.env.PORT || 3000);
if (require.main === module) {
  app.listen(PORT, () => {
    console.log(`Telegram Bot & API listening on port ${PORT} (backend=${backendBaseUrl})`);
    if (bot) {
      bot.start().catch((err: Error) =>
        console.warn("Bot polling failed to start:", err.message),
      );
    }
  });
}

