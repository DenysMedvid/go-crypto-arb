import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { SummaryCard } from './SummaryCard';

describe('SummaryCard', () => {
  it('renders dashboard summary values', () => {
    render(<SummaryCard title="Alerts" value={3} detail="Review active alerts" tone="warning" />);

    expect(screen.getByLabelText('Alerts')).toHaveTextContent('3');
    expect(screen.getByText('Review active alerts')).toBeInTheDocument();
  });
});
