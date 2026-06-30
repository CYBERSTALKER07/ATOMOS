import type { ReactNode } from "react";

export type PropRow = {
  name: string;
  type: string;
  default?: string;
  description: string;
};

export type ComponentDoc = {
  slug: string;
  title: string;
  description: string;
  props: PropRow[];
  motionSpec: string;
  usedIn: { role: string; surface: string; href?: string }[];
  snippet: string;
  Preview: () => ReactNode;
};

export type ComponentDocMeta = Omit<ComponentDoc, "Preview">;
