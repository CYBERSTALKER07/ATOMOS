"use client";

import {
  X,
  ShoppingBag,
  CreditCard,
  ChevronRight,
  Minus,
  Plus,
  Trash2,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useCart } from "../lib/cart";

interface CartDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  onCheckout: () => void;
}

export default function CartDrawer({
  isOpen,
  onClose,
  onCheckout,
}: CartDrawerProps) {
  const { items, updateQuantity, removeFromCart, total } = useCart();

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50 overflow-hidden">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-[#0a0a0a]/60 backdrop-blur-sm"
            onClick={onClose}
          />

          <motion.div
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", stiffness: 400, damping: 40 }}
            className="absolute top-0 right-0 w-full max-w-[440px] h-full shadow-2xl flex flex-col border-l border-[var(--desk-border)] bg-[var(--desk-surface)]"
          >
            {/* Header */}
            <div className="p-6 flex justify-between items-center border-b border-[var(--desk-border)]">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center">
                  <ShoppingBag size={22} />
                </div>
                <div>
                  <h2 className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                    Staged Assets
                  </h2>
                  <p className="text-[10px] font-bold text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                    {items.length} Nodes in Queue
                  </p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="w-10 h-10 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
              >
                <X size={20} />
              </button>
            </div>

            {/* List */}
            <div className="flex-1 overflow-y-auto p-6 space-y-4">
              <AnimatePresence mode="popLayout">
                {items.map((item) => (
                  <motion.div
                    key={item.product_id}
                    layout
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.95 }}
                    className="p-4 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] rounded-2xl flex gap-4 group"
                  >
                    <div className="w-16 h-16 rounded-xl bg-[var(--desk-surface)] border border-[var(--desk-border)] flex-shrink-0 flex items-center justify-center overflow-hidden">
                      {item.image_url ? (
                        <img
                          src={item.image_url}
                          alt={item.name}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <Package
                          className="text-[var(--desk-text-tertiary)] opacity-20"
                          size={24}
                        />
                      )}
                    </div>
                    <div className="flex-1 min-w-0 flex flex-col justify-between">
                      <div className="flex items-start justify-between gap-2">
                        <h4 className="md-typescale-title-small font-bold text-[var(--desk-text-primary)] line-clamp-1">
                          {item.name}
                        </h4>
                        <button
                          onClick={() => removeFromCart(item.product_id)}
                          className="text-[var(--desk-text-tertiary)] hover:text-[var(--desk-danger)] transition-colors p-1"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>

                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2 bg-[var(--desk-canvas)] p-1 rounded-lg">
                          <button
                            onClick={() =>
                              updateQuantity(item.product_id, item.quantity - 1)
                            }
                            className="w-6 h-6 rounded-md bg-[var(--desk-surface)] flex items-center justify-center shadow-sm active:scale-90 transition-all"
                          >
                            <Minus size={12} />
                          </button>
                          <span className="md-typescale-label-small font-bold w-4 text-center tabular-nums">
                            {item.quantity}
                          </span>
                          <button
                            onClick={() =>
                              updateQuantity(item.product_id, item.quantity + 1)
                            }
                            className="w-6 h-6 rounded-md bg-[var(--desk-surface)] flex items-center justify-center shadow-sm active:scale-90 transition-all"
                          >
                            <Plus size={12} />
                          </button>
                        </div>
                        <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                          {(item.price * item.quantity).toLocaleString()}
                        </span>
                      </div>
                    </div>
                  </motion.div>
                ))}
                {items.length === 0 && (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="py-20 text-center opacity-40"
                  >
                    <ShoppingBag size={48} className="mx-auto mb-4" />
                    <p className="md-typescale-body-large">Cart is empty</p>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            {/* Footer */}
            <div className="p-8 border-t border-[var(--desk-border)] bg-[var(--desk-surface-subtle)]">
              <div className="space-y-2 mb-8">
                <div className="flex justify-between md-typescale-label-small text-[var(--desk-text-tertiary)] uppercase font-bold tracking-widest">
                  <span>Operational Subtotal</span>
                  <span className="text-[var(--desk-text-secondary)]">
                    {total.toLocaleString()}
                  </span>
                </div>
                <div className="flex justify-between items-end">
                  <span className="md-typescale-title-large font-bold text-[var(--desk-text-primary)]">
                    Total Settlement
                  </span>
                  <span className="md-typescale-display-small font-bold text-[var(--desk-text-primary)] tabular-nums">
                    {total.toLocaleString()}{" "}
                    <small className="text-xs opacity-40 ml-0.5">UZS</small>
                  </span>
                </div>
              </div>

              <Button
                disabled={items.length === 0}
                onPress={() => onCheckout()}
                className="w-full h-14 rounded-2xl flex items-center justify-center gap-3 bg-[var(--desk-text-primary)] text-white font-bold shadow-xl transition-all hover:scale-[1.02] active:scale-95 disabled:opacity-30 disabled:hover:scale-100"
              >
                <CreditCard size={20} />
                Execute Procurement
                <ChevronRight size={20} />
              </Button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
