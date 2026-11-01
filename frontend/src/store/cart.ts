import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Product } from '../types';

export interface CartLine {
  productId: string;
  name: string;
  price: number;
  imageUrl: string;
  quantity: number;
}

interface CartState {
  lines: CartLine[];
  add: (product: Product, quantity?: number) => void;
  setQuantity: (productId: string, quantity: number) => void;
  remove: (productId: string) => void;
  clear: () => void;
}

export const useCartStore = create<CartState>()(
  persist(
    (set) => ({
      lines: [],
      add: (product, quantity = 1) =>
        set((state) => {
          const existing = state.lines.find((line) => line.productId === product.id);
          if (existing) {
            return {
              lines: state.lines.map((line) =>
                line.productId === product.id ? { ...line, quantity: Math.min(line.quantity + quantity, 20) } : line,
              ),
            };
          }
          return {
            lines: [
              ...state.lines,
              { productId: product.id, name: product.name, price: product.price, imageUrl: product.imageUrl, quantity },
            ],
          };
        }),
      setQuantity: (productId, quantity) =>
        set((state) => ({
          lines: state.lines.map((line) =>
            line.productId === productId ? { ...line, quantity: Math.max(1, Math.min(quantity, 20)) } : line,
          ),
        })),
      remove: (productId) => set((state) => ({ lines: state.lines.filter((line) => line.productId !== productId) })),
      clear: () => set({ lines: [] }),
    }),
    { name: 'mongoscale-cart' },
  ),
);

export function cartTotal(lines: CartLine[]): number {
  return lines.reduce((sum, line) => sum + line.price * line.quantity, 0);
}
