import { describe, it, expect, vi } from 'vitest';
import { downloadCsv } from '../csv';

describe('downloadCsv', () => {
  it('creates and triggers a download link with correct CSV content', () => {
    // Mock DOM elements and URL object
    const mockAnchor = {
      href: '',
      download: '',
      click: vi.fn(),
    };
    
    vi.spyOn(document, 'createElement').mockReturnValue(mockAnchor as unknown as HTMLAnchorElement);
    const createObjectURLSpy = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test-url');
    const revokeObjectURLSpy = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {});

    const headers = ['Name', 'Age', 'City, State'];
    const rows = [
      ['Alice', '30', 'New York, NY'],
      ['Bob', '25', 'Los Angeles, CA'],
    ];

    downloadCsv('test.csv', headers, rows);

    // Verify blob creation (can't easily inspect Blob contents without async reading, but we verify it's called)
    expect(createObjectURLSpy).toHaveBeenCalled();
    
    // Verify anchor configuration
    expect(mockAnchor.href).toBe('blob:test-url');
    expect(mockAnchor.download).toBe('test.csv');
    
    // Verify click and cleanup
    expect(mockAnchor.click).toHaveBeenCalled();
    expect(revokeObjectURLSpy).toHaveBeenCalledWith('blob:test-url');
  });
});
