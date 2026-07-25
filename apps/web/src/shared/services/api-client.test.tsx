import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/renderWithProviders';

describe('test harness', () => {
  it('renders a component through the providers', () => {
    const { getByText } = renderWithProviders(<span>harness ok</span>);
    expect(getByText('harness ok')).toBeInTheDocument();
  });
});
