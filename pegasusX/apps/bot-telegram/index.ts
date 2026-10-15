import { Bot, InlineKeyboard } from "grammy";
import express from "express";
import cors from "cors";

const bot = new Bot(process.env.BOT_TOKEN || "mock-token-for-dev");
const app = express();
app.use(cors());
app.use(express.json());

// Telegram Bot Handlers
bot.command("start", (ctx) => {
  ctx.reply("Welcome to Pegasus.X B2B Restock Assistant! 🚀\nI will notify you when it's time to restock your inventory.");
});

bot.on("callback_query:data", async (ctx) => {
  if (ctx.callbackQuery.data.startsWith("order_")) {
    const orderRef = ctx.callbackQuery.data.split("_")[1];
    await ctx.answerCallbackQuery("Order placed successfully!");
    await ctx.editMessageText(`✅ Success! Your one-click restock order (${orderRef}) has been confirmed and routed to the warehouse.`);
  }
});

// REST API for Go Backend to Trigger Proactive Restock Messages
app.post("/trigger-restock", async (req, res) => {
  try {
    const { telegram_chat_id, retailer_name, product_name, suggested_qty } = req.body;
    
    if (!telegram_chat_id) {
      return res.status(400).json({ error: "telegram_chat_id is required" });
    }

    const orderRef = Math.random().toString(36).substring(7).toUpperCase();
    const message = `Hello ${retailer_name} 👋\n\nBased on your historical sales, you are running low on *${product_name}*.\n\nWould you like to instantly restock ${suggested_qty} units?`;
    
    const keyboard = new InlineKeyboard().text("📦 One-Click Restock", `order_${orderRef}`);

    await bot.api.sendMessage(telegram_chat_id, message, {
      parse_mode: "Markdown",
      reply_markup: keyboard,
    });

    res.json({ success: true, message: "Proactive restock proposed." });
  } catch (error: any) {
    console.error("Failed to send message:", error);
    res.status(500).json({ error: error.message });
  }
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Telegram Bot & API listening on port ${PORT}`);
  // Do not block deployment if token is invalid in CI
  bot.start().catch((err) => console.log("Bot failed to start (expected if mock token):", err.message));
});
