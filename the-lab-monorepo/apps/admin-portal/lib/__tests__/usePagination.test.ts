import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePagination } from '../usePagination';

describe('usePagination', () => {
  const items = Array.from({ length: 100 }, (_, i) => `item-${i}`);

  it('defaults to page 1 with 25 items per page', () => {
    const { result } = renderHook(() => usePagination(items));
    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(25);
    expect(result.current.pageItems.length).toBe(25);
    expect(result.current.totalPages).toBe(4);
    expect(result.current.totalItems).toBe(100);
  });

  it('respects custom defaultPageSize', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    expect(result.current.pageSize).toBe(10);
    expect(result.current.totalPages).toBe(10);
    expect(result.current.pageItems.length).toBe(10);
  });

  it('nextPage advances to page 2', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    act(() => result.current.nextPage());
    expect(result.current.page).toBe(2);
    expect(result.current.pageItems[0]).toBe('item-10');
  });

  it('prevPage goes back', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    act(() => result.current.nextPage());
    act(() => result.current.nextPage());
    expect(result.current.page).toBe(3);
    act(() => result.current.prevPage());
    expect(result.current.page).toBe(2);
  });

  it('prevPage does not go below 1', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    act(() => result.current.prevPage());
    expect(result.current.page).toBe(1);
  });

  it('nextPage does not go past totalPages', () => {
    const { result } = renderHook(() => usePagination(items, 50));
    expect(result.current.totalPages).toBe(2);
    act(() => result.current.nextPage());
    act(() => result.current.nextPage());
    expect(result.current.page).toBe(2);
  });

  it('canNext and canPrev flags are correct', () => {
    const { result } = renderHook(() => usePagination(items, 50));
    expect(result.current.canNext).toBe(true);
    expect(result.current.canPrev).toBe(false);
    act(() => result.current.nextPage());
    expect(result.current.canNext).toBe(false);
    expect(result.current.canPrev).toBe(true);
  });

  it('setPage jumps to specific page', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    act(() => result.current.setPage(5));
    expect(result.current.page).toBe(5);
    expect(result.current.pageItems[0]).toBe('item-40');
  });

  it('setPageSize resets to page 1', () => {
    const { result } = renderHook(() => usePagination(items, 10));
    act(() => result.current.setPage(3));
    act(() => result.current.setPageSize(20));
    expect(result.current.page).toBe(1);
    expect(result.current.pageSize).toBe(20);
    expect(result.current.totalPages).toBe(5);
  });

  it('handles empty items', () => {
    const { result } = renderHook(() => usePagination([]));
    expect(result.current.totalPages).toBe(1);
    expect(result.current.pageItems.length).toBe(0);
    expect(result.current.canNext).toBe(false);
    expect(result.current.canPrev).toBe(false);
  });

  it('clamps page when data shrinks', () => {
    let data = items;
    const { result, rerender } = renderHook(() => usePagination(data, 10));
    act(() => result.current.setPage(10)); // last page of 100 items
    expect(result.current.page).toBe(10);
    // Shrink to 30 items
    data = items.slice(0, 30);
    rerender();
    expect(result.current.page).toBe(3); // clamped to 3 (totalPages for 30 items / 10)
  });

  it('last page has correct remainder items', () => {
    const odd = Array.from({ length: 23 }, (_, i) => i);
    const { result } = renderHook(() => usePagination(odd, 10));
    act(() => result.current.setPage(3));
    expect(result.current.pageItems.length).toBe(3);
  });
});
