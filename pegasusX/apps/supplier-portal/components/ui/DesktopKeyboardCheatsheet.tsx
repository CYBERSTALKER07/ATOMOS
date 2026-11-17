"use client";

import React, { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, Command, Keyboard } from "lucide-react";

export type ShortcutGroup = {
  category: string;
  shortcuts: {
    keys: string[];
    description: string;
  }[];
};

const DEFAULT_SHORTCUTS: ShortcutGroup[] = [
  {
    category: "Global Navigation",
    shortcuts: [
      { keys: ["⌘", "K"], description: "Open Command Palette / Search" },
      { keys: ["?"], description: "Toggle Keyboard Shortcuts Cheatsheet" },
      { keys: ["Esc"], description: "Close active modal, drawer, or search" },
    ],
  },
  {
    category: "Foundry & Scenarios",
    shortcuts: [
      { keys: ["⌘", "↵"], description: "Stage / Apply Copy-on-Write patch" },
      { keys: ["⌘", "S"], description: "Commit Scenario to live Spanner" },
      { keys: ["⌥", "R"], description: "Run SearchAround / Lot Recall" },
    ],
  },
  {
    category: "System & Connectivity",
    shortcuts: [
      { keys: ["⌘", "R"], description: "Refresh telemetry & SWR cache" },
      { keys: ["⌥", "O"], description: "Toggle Offline Queue inspection" },
    ],
  },
];

export interface DesktopKeyboardCheatsheetProps {
  open: boolean;
  onClose: () => void;
  groups?: ShortcutGroup[];
}

export function DesktopKeyboardCheatsheet({
  open,
  onClose,
  groups = DEFAULT_SHORTCUTS,
}: DesktopKeyboardCheatsheetProps) {
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onClose]);

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm select-none">
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 10 }}
            transition={{ type: "spring", stiffness: 350, damping: 28 }}
            className="w-full max-w-xl rounded-2xl border border-neutral-800 bg-neutral-900/95 p-6 text-neutral-100 shadow-2xl backdrop-blur-xl"
            role="dialog"
            aria-modal="true"
            aria-labelledby="shortcuts-title"
          >
            <div className="flex items-center justify-between border-b border-neutral-800 pb-4">
              <div className="flex items-center gap-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-750 bg-neutral-800 text-neutral-300">
                  <Keyboard size={18} />
                </div>
                <div>
                  <h2 id="shortcuts-title" className="text-sm font-bold tracking-wide text-neutral-100 uppercase">
                    Desktop Keyboard Shortcuts
                  </h2>
                  <p className="text-xs text-neutral-400">
                    High-efficiency hotkeys for the PegasusX Supplier Portal
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={onClose}
                className="flex h-7 w-7 items-center justify-center rounded-lg border border-neutral-800 text-neutral-400 hover:border-neutral-700 hover:bg-neutral-800 hover:text-neutral-100 transition-colors"
                aria-label="Close shortcuts"
              >
                <X size={16} />
              </button>
            </div>

            <div className="mt-5 space-y-6 max-h-[60vh] overflow-y-auto pr-1">
              {groups.map((group) => (
                <div key={group.category} className="space-y-2">
                  <h3 className="font-mono text-[11px] font-semibold uppercase tracking-wider text-neutral-400">
                    {group.category}
                  </h3>
                  <div className="divide-y divide-neutral-800/60 rounded-xl border border-neutral-800 bg-neutral-950/60 overflow-hidden">
                    {group.shortcuts.map((sc, idx) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between px-3.5 py-2.5 text-xs hover:bg-neutral-800/30 transition-colors"
                      >
                        <span className="text-neutral-300 font-medium">{sc.description}</span>
                        <div className="flex items-center gap-1 font-mono">
                          {sc.keys.map((k, ki) => (
                            <kbd
                              key={ki}
                              className="min-w-6 inline-flex h-5 items-center justify-center rounded border border-neutral-700 bg-neutral-800 px-1.5 text-[11px] font-semibold text-neutral-200 shadow-sm"
                            >
                              {k === "⌘" ? <Command size={11} className="mr-0.5" /> : null}
                              {k}
                            </kbd>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-6 flex items-center justify-between border-t border-neutral-800/80 pt-4 text-[11px] font-mono text-neutral-500">
              <span>PegasusX Desktop v2.4 (Tauri Runtime)</span>
              <span>Press <kbd className="rounded border border-neutral-700 bg-neutral-800 px-1 text-neutral-300">Esc</kbd> to dismiss</span>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
