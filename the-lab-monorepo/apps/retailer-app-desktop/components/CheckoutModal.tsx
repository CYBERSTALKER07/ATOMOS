'use client';

import { useState } from "react";
import { X, Building2, Ticket, CreditCard, Loader2 } from "lucide-react";
import { useCart } from "../lib/cart";
import { apiFetch } from "../lib/auth";
import { useRouter } from "next/navigation";
import type { UnifiedCheckoutResponse, RetailerProfile } from "../lib/types";

function getProfile(): RetailerProfile | null {
  if (typeof localStorage === 'undefined') return null;
  try {
    const raw = localStorage.getItem('retailer_profile');
    return raw ? JSON.parse(raw) : null;
  } catch { return null; }
}

interface CheckoutModalProps {
  isOpen: boolean;
  onClose: () => void;
  total: number;
}

export default function CheckoutModal({ isOpen, onClose, total }: CheckoutModalProps) {
  const { items, clearCart } = useCart();
  const [method, setMethod] = useState<'global_pay'|'invoice'|'cash'>('invoice');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [oosItems, setOosItems] = useState<string[]>([]);
  const router = useRouter();

  const handleCheckout = async () => {
    if (items.length === 0) return;
    setLoading(true);
    setError('');
    setOosItems([]);

    try {
      const profile = getProfile();
      if (!profile?.id) throw new Error('Retailer profile not found. Please log in again.');

      // Map payment method to backend gateway name
      const gatewayMap: Record<string, string> = {
        invoice: 'BANK_TRANSFER',
        cash: 'CASH',
        global_pay: 'GLOBAL_PAY',
      };

      // Build line items matching backend UnifiedCheckoutRequest shape
      const lineItems = items.map(item => ({
        sku_id: item.product_id,
        quantity: item.quantity,
        unit_price: item.price,
      }));

      // 1. Fan-out Cart via unified checkout
      const cartRes = await apiFetch('/v1/checkout/unified', {
        method: 'POST',
        body: JSON.stringify({
          retailer_id: profile.id,
          payment_gateway: gatewayMap[method] || 'BANK_TRANSFER',
          latitude: 0,
          longitude: 0,
          items: lineItems,
        }),
      });

      if (!cartRes.ok) {
        const errBody = await cartRes.json().catch(() => null);
        if (cartRes.status === 409 && errBody?.code === 'ALL_ITEMS_OUT_OF_STOCK') {
          setOosItems(errBody.oos_items || []);
          throw new Error('All items are out of stock. Please update your cart.');
        }
        throw new Error(errBody?.error || 'Failed to create orders from cart');
      }
      const cartData: UnifiedCheckoutResponse = await cartRes.json();
      const supplierOrders = cartData.supplier_orders || [];

      // 2. Initiate payment for each supplier order
      if (['global_pay'].includes(method)) {
        for (const so of supplierOrders) {
          const payRes = await apiFetch('/v1/order/card-checkout', {
            method: 'POST',
            body: JSON.stringify({
              order_id: so.order_id,
              gateway: gatewayMap[method],
              amount: so.total,
              return_url: 'retailer-app://orders',
            }),
          });
          if (!payRes.ok) throw new Error(`Payment initiation failed for order ${so.order_id}`);
          const payData = await payRes.json();
          if (payData.payment_url) {
            window.open(payData.payment_url, '_blank');
          }
        }
      } else if (method === 'cash') {
        for (const so of supplierOrders) {
          await apiFetch('/v1/order/cash-checkout', {
            method: 'POST',
            body: JSON.stringify({ order_id: so.order_id }),
          });
        }
      }
      // invoice method: no extra payment step needed — orders created with BANK_TRANSFER

      clearCart();
      onClose();
      router.push('/orders');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Checkout failed');
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      
      <div className="relative w-full max-w-2xl rounded-[28px] shadow-2xl flex flex-col max-h-[90vh] overflow-hidden transform animate-fade-in-up" style={{ background: 'var(--background)' }}>
        
        <div className="px-8 py-6 border-b border-[var(--border)] flex items-center justify-between" style={{ background: 'var(--background)' }}>
          <h2 className="md-typescale-headline-small font-bold text-foreground">
            Checkout & Payment
          </h2>
          <button 
            onClick={onClose} 
            className="w-10 h-10 flex items-center justify-center rounded-full text-muted hover:text-foreground transition-all" style={{ background: 'var(--surface)' }}
          >
            <X size={20} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-8 flex flex-col gap-8" style={{ background: 'var(--background)' }}>
          {error && (
            <div className="p-4 rounded-xl" style={{ background: 'rgba(220,38,38,0.08)', color: 'var(--danger)' }}>
              <p>{error}</p>
              {oosItems.length > 0 && (
                <ul className="mt-2 text-sm opacity-80 list-disc pl-4">
                  {oosItems.map(sku => <li key={sku}>{sku}</li>)}
                </ul>
              )}
            </div>
          )}

          <div className="p-6 rounded-2xl border border-[var(--border)] flex items-center justify-between" style={{ background: 'var(--surface)' }}>
            <div>
              <p className="md-typescale-body-medium text-muted uppercase tracking-wider font-semibold mb-1">Total Due</p>
              <h3 className="md-typescale-display-small font-bold" style={{ color: 'var(--accent)' }}>{total.toLocaleString()}</h3>
            </div>
            <div className="text-right">
              <p className="md-typescale-body-medium text-muted mb-1">Delivery Estimate</p>
              <p className="md-typescale-title-medium font-semibold text-foreground">Tomorrow</p>
            </div>
          </div>

          <div>
            <h3 className="md-typescale-title-medium font-bold text-foreground mb-4">
              Select Payment Gateway
            </h3>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div 
                onClick={() => setMethod('cash')}
                className={`border p-5 rounded-2xl cursor-pointer transition-all flex items-center gap-4 ${method === 'cash' ? 'border-[var(--warning)] bg-[var(--warning)]/8' : 'border-[var(--border)] hover:border-[var(--warning)] hover:bg-[var(--warning)]/4'}`}
              >
                <div className="w-12 h-12 rounded-xl flex items-center justify-center text-muted" style={{ background: 'var(--surface)' }}>
                  <Ticket size={24} />
                </div>
                <div>
                  <h4 className="md-typescale-title-medium font-bold text-foreground">Cash on Delivery</h4>
                  <p className="md-typescale-body-small text-muted flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full" style={{ background: 'var(--warning)' }} />
                    Available for overrides
                  </p>
                </div>
              </div>

              <div 
                onClick={() => setMethod('global_pay')}
                className={`border p-5 rounded-2xl cursor-pointer transition-all flex items-center gap-4 ${method === 'global_pay' ? 'border-[var(--accent)] bg-[var(--accent)]/5' : 'border-[var(--border)] hover:border-[var(--accent)] hover:bg-[var(--accent)]/3'}`}
              >
                <div className="w-12 h-12 bg-cyan-500/10 text-cyan-600 rounded-xl flex items-center justify-center">
                  <CreditCard size={24} />
                </div>
                <div>
                  <h4 className="md-typescale-title-medium font-bold text-foreground">Global Pay</h4>
                  <p className="md-typescale-body-small text-muted flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full" style={{ background: 'var(--success)' }} />
                    Credit & Debit Cards
                  </p>
                </div>
              </div>

              <div 
                onClick={() => setMethod('invoice')}
                className={`border p-5 rounded-2xl cursor-pointer transition-all flex items-center gap-4 relative overflow-hidden ${method === 'invoice' ? 'border-[var(--accent)] bg-[var(--accent)]/5' : 'border-[var(--border)]'}`}
              >
                {method === 'invoice' && <div className="absolute top-0 right-0 w-2 h-full" style={{ background: 'var(--accent)' }} />}
                <div className="w-12 h-12 rounded-xl flex items-center justify-center" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                  <Building2 size={24} />
                </div>
                <div>
                  <h4 className="md-typescale-title-medium font-bold text-foreground">Invoice / Net 30</h4>
                  <p className="md-typescale-body-small text-muted">Pre-approved corporate line</p>
                </div>
              </div>
              
              <div 
                onClick={() => setMethod('cash')}
                className={`border p-5 rounded-2xl cursor-pointer transition-all flex items-center gap-4 ${method === 'cash' ? 'border-[var(--warning)] bg-[var(--warning)]/8' : 'border-[var(--border)] hover:border-[var(--warning)] hover:bg-[var(--warning)]/4'}`}
              >
                <div className="w-12 h-12 rounded-xl flex items-center justify-center text-muted" style={{ background: 'var(--surface)' }}>
                  <Ticket size={24} />
                </div>
                <div>
                  <h4 className="md-typescale-title-medium font-bold text-foreground">Cash on Delivery</h4>
                  <p className="md-typescale-body-small text-muted flex items-center gap-1">
                    <span className="w-2 h-2 rounded-full" style={{ background: 'var(--warning)' }} />
                    Available for overrides
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="px-8 py-6 border-t border-[var(--border)] flex justify-end gap-3" style={{ background: 'var(--background)' }}>
          <button 
            onClick={onClose}
            disabled={loading}
            className="md-btn px-6 py-2.5 rounded-full font-bold text-muted hover:bg-surface transition-colors"
          >
            Cancel
          </button>
          <button 
            onClick={handleCheckout}
            disabled={loading || items.length === 0}
            className="md-btn md-btn-filled flex items-center justify-center gap-2 px-8 py-2.5 rounded-full font-bold hover:opacity-90 transition-opacity disabled:opacity-50"
            style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-foreground)' }}
          >
            {loading ? <Loader2 size={18} className="animate-spin" /> : 'Confirm & Pay'}
          </button>
        </div>

      </div>
    </div>
  );
}
