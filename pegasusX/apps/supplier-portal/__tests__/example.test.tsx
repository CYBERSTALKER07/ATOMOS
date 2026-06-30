import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import React from 'react';

describe('Supplier Portal Example', () => {
  it('renders successfully', () => {
    render(<div>Supplier Portal</div>);
    expect(screen.getByText('Supplier Portal')).toBeInTheDocument();
  });
});
