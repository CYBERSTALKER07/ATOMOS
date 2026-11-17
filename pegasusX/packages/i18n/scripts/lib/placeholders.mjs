/**
 * Canonical catalog placeholders are {name} (see translationContract).
 * Native Android/iOS resources use positional printf-style specs so
 * translators can reorder and stringResource / String(format:) work.
 */

const NAMED = /\{([a-zA-Z0-9_]+)\}/g;

/** Ordered unique placeholder names as they first appear. */
export function listPlaceholders(value) {
  const order = [];
  const seen = new Set();
  for (const match of value.matchAll(NAMED)) {
    const name = match[1];
    if (seen.has(name)) continue;
    seen.add(name);
    order.push(name);
  }
  return order;
}

/**
 * Convert "Hello {name}, {count}" → Android "%1$s" / iOS "%1$@" form.
 * Literal % in static text become %%. Repeated {name} reuse the same index.
 */
export function namedToPositional(value, platform) {
  const indexByName = new Map();
  let next = 1;

  // Split into static / {name} runs so we can escape % only in static runs.
  const parts = [];
  let last = 0;
  for (const match of value.matchAll(NAMED)) {
    parts.push({ type: "static", text: value.slice(last, match.index) });
    parts.push({ type: "ph", name: match[1] });
    last = match.index + match[0].length;
  }
  parts.push({ type: "static", text: value.slice(last) });

  return parts
    .map((part) => {
      if (part.type === "static") {
        return part.text.replace(/%/g, "%%");
      }
      if (!indexByName.has(part.name)) {
        indexByName.set(part.name, next);
        next += 1;
      }
      const n = indexByName.get(part.name);
      return platform === "android" ? `%${n}$s` : `%${n}$@`;
    })
    .join("");
}

/** Mask {placeholders} so draft translators cannot scramble them. */
export function mapPlaceholders(value, fn) {
  const held = [];
  const masked = value.replace(NAMED, (m) => {
    held.push(m);
    return `\uE000${held.length - 1}\uE001`;
  });
  const mapped = fn(masked);
  return mapped.replace(/\uE000(\d+)\uE001/g, (_, i) => held[Number(i)]);
}
