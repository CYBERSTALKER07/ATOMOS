import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { KpiStatCard, KpiStatGrid } from '../KpiStatCard';

describe('KpiStatCard', () => {
  it('renders the label and value correctly', () => {
    render(<KpiStatCard label="Total Revenue" value="$10,000" />);
    
    expect(screen.getByText('Total Revenue')).toBeInTheDocument();
    expect(screen.getByText('$10,000')).toBeInTheDocument();
  });

  it('renders the subtext if provided', () => {
    render(<KpiStatCard label="Active Users" value="1,200" sub="+5% from last week" />);
    
    expect(screen.getByText('Active Users')).toBeInTheDocument();
    expect(screen.getByText('1,200')).toBeInTheDocument();
    expect(screen.getByText('+5% from last week')).toBeInTheDocument();
  });
});

describe('KpiStatGrid', () => {
  it('renders children correctly', () => {
    render(
      <KpiStatGrid>
        <div data-testid="child-1" />
        <div data-testid="child-2" />
      </KpiStatGrid>
    );

    expect(screen.getByTestId('child-1')).toBeInTheDocument();
    expect(screen.getByTestId('child-2')).toBeInTheDocument();
  });
});
