import { describe, expect, it } from "vitest";
import {
  TIYIN_PER_UZS,
  createTranslator,
  defaultLocale,
  formatUZS,
  getTranslation,
  messageCatalogs,
  resolveLocale,
  supportedLocales,
  translationContract,
} from "../index";
import { formatUZS as formatUZSDirect } from "../money";
import { readAllFlatCatalogs, validateCatalogs } from "../scripts/shared.mjs";

describe("@pegasusx/i18n locales baseline", () => {
  it("uses uz as defaultLocale while supporting en, ru, and uz", () => {
    expect(defaultLocale).toBe("uz");
    expect([...supportedLocales]).toEqual(["en", "ru", "uz"]);
    expect(resolveLocale(undefined)).toBe("uz");
    expect(resolveLocale(null)).toBe("uz");
    expect(resolveLocale("fr-FR")).toBe("uz");
    expect(resolveLocale("ru")).toBe("ru");
    expect(resolveLocale("en")).toBe("en");
  });

  it("falls back to uz default catalog when a requested locale omits a key", () => {
    const catalogs = {
      en: { greeting: "Hello" },
      ru: {},
      uz: { greeting: "Salom" },
    };
    expect(getTranslation(catalogs, "ru", "greeting")).toBe("Salom");
    expect(getTranslation(catalogs, "en", "greeting")).toBe("Hello");

    const sampleKey = Object.keys(messageCatalogs.uz).find(
      (k) => !messageCatalogs.uz[k].includes("{") && messageCatalogs.uz[k].length > 0,
    );
    expect(sampleKey).toBeDefined();
    const defaultT = createTranslator(undefined);
    expect(defaultT(sampleKey!)).toBe(messageCatalogs.uz[sampleKey!]);
  });
});

describe("@pegasusx/i18n strict formatUZS", () => {
  it("converts minor units (tiyin) to integer so'm (100 tiyin = 1 so'm) and never renders raw tiyin", () => {
    expect(TIYIN_PER_UZS).toBe(BigInt(100));
    expect(formatUZS).toBe(formatUZSDirect);

    expect(formatUZS(0)).toBe("0 so'm");
    expect(formatUZS(-0)).toBe("0 so'm");
    expect(formatUZS(99)).toBe("0 so'm");
    expect(formatUZS(-99)).toBe("0 so'm");
    expect(formatUZS(100)).toBe("1 so'm");
    expect(formatUZS(199)).toBe("1 so'm");
    expect(formatUZS(150000)).toBe("1 500 so'm");
    expect(formatUZS(142500000)).toBe("1 425 000 so'm");
    expect(formatUZS(-250500)).toBe("-2 505 so'm");
  });

  it("supports exact BigInt minor values beyond Number.MAX_SAFE_INTEGER", () => {
    expect(formatUZS(BigInt(100))).toBe("1 so'm");
    expect(formatUZS(BigInt("900719925474099300"))).toBe("9 007 199 254 740 993 so'm");
    expect(formatUZS(BigInt("-123456789012345678900"))).toBe("-1 234 567 890 123 456 789 so'm");
  });

  it("strictly rejects fractional, non-finite, unsafe integers, and non-numeric inputs", () => {
    expect(() => formatUZS(150.5)).toThrow(RangeError);
    expect(() => formatUZS(0.01)).toThrow(RangeError);
    expect(() => formatUZS(Number.NaN)).toThrow(RangeError);
    expect(() => formatUZS(Number.POSITIVE_INFINITY)).toThrow(RangeError);
    expect(() => formatUZS(Number.NEGATIVE_INFINITY)).toThrow(RangeError);
    expect(() => formatUZS(Number.MAX_SAFE_INTEGER + 1)).toThrow(RangeError);

    expect(() => formatUZS("100" as unknown as number)).toThrow(TypeError);
    expect(() => formatUZS(null as unknown as number)).toThrow(TypeError);
    expect(() => formatUZS(undefined as unknown as number)).toThrow(TypeError);
  });
});

describe("@pegasusx/i18n strict D-11 catalog parity and W5 keys", () => {
  it("enforces defaultLocale=uz, uzbek latin script, and zero untranslated multi-word English across ru/uz", () => {
    expect(translationContract.defaultLocale).toBe("uz");
    expect(translationContract.uzbekScript).toBe("latin");
    expect(translationContract.fallbackChain[0]).toBe("uz");

    const flat = readAllFlatCatalogs();
    const validation = validateCatalogs(flat);
    expect(validation.keyCount).toBeGreaterThanOrEqual(4800);

    const enKeys = Object.keys(messageCatalogs.en);
    expect(Object.keys(messageCatalogs.ru).length).toBe(enKeys.length);
    expect(Object.keys(messageCatalogs.uz).length).toBe(enKeys.length);
  });

  it("includes localized W5 Fast Mode, Exception Desk, SyncQueue, and Telegram Bot keys without raw minor/JWT prompts", () => {
    const requiredW5Keys = [
      "retailer_desktop.fast_mode.tomorrows_delivery",
      "retailer_desktop.fast_mode.owed_balance",
      "retailer_desktop.fast_mode.todays_proposal",
      "retailer_desktop.fast_mode.auto_confirm_before_cutoff",
      "retailer_desktop.fast_mode.cutoff_countdown",
      "retailer_desktop.fast_mode.switch_to_full_erp",
      "retailer_desktop.fast_mode.switch_to_fast_mode",
      "supplier_portal.exception_desk.title",
      "supplier_portal.exception_desk.money_at_risk",
      "supplier_portal.exception_desk.batch_resolve",
      "supplier_portal.exception_desk.severity.critical",
      "supplier_portal.exception_desk.severity.high",
      "supplier_portal.exception_desk.severity.medium",
      "supplier_portal.exception_desk.severity.low",
      "supplier_portal.exception_desk.handlers.fiscal_and_export_blockers",
      "supplier_portal.exception_desk.handlers.cash_discrepancy",
      "supplier_portal.exception_desk.handlers.claims_review",
      "supplier_portal.exception_desk.handlers.route_and_closed_door",
      "supplier_portal.exception_desk.handlers.credit_and_promise",
      "supplier_portal.exception_desk.handlers.fulfillment_and_catalog",
      "mobile_driver.sync_queue.queued_for_soliq_retry",
      "mobile_driver.sync_queue.escalate_to_support_review",
      "mobile_driver.sync_queue.retry_queued_items",
      "telegram_bot.welcome.title",
      "telegram_bot.bind.share_phone_button",
      "telegram_bot.bind.enter_otp_prompt",
      "telegram_bot.orders.line",
      "telegram_bot.debt.balance_owed",
      "telegram_bot.lang.updated",
      "telegram_bot.restock.prompt",
    ];

    for (const key of requiredW5Keys) {
      expect(messageCatalogs.en[key]).toBeTruthy();
      expect(messageCatalogs.ru[key]).toBeTruthy();
      expect(messageCatalogs.uz[key]).toBeTruthy();
    }

    const uzHelp = createTranslator("uz")("telegram_bot.help.commands");
    expect(uzHelp).toContain("/link <6-xonali kod>");
    expect(uzHelp).not.toContain("jwt");
  });
});
