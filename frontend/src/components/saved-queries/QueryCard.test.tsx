/**
 * QueryCard Component Tests
 *
 * Basic test suite for QueryCard component functionality
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { QueryCard } from './QueryCard'
import type { SavedQueryRecord } from '@/types/storage'

// Mock date-fns to avoid timezone issues in tests
vi.mock('date-fns', () => ({
  formatDistanceToNow: () => '2 hours ago',
}))

describe('QueryCard', () => {
  const mockQuery: SavedQueryRecord = {
    id: 'test-query-1',
    user_id: 'user-1',
    title: 'Top Users Query',
    description: 'Get the top 10 users by revenue for the current month',
    query_text: 'SELECT * FROM users ORDER BY revenue DESC LIMIT 10',
    tags: ['analytics', 'users'],
    folder: 'Revenue Reports',
    is_favorite: true,
    created_at: new Date('2025-01-01'),
    updated_at: new Date('2025-01-20'),
    synced: true,
    sync_version: 1,
  }

  const mockHandlers = {
    onLoad: vi.fn(),
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onDuplicate: vi.fn(),
    onToggleFavorite: vi.fn(),
  }

  it('renders query title and description', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    expect(screen.getByText('Top Users Query')).toBeInTheDocument()
    expect(
      screen.getByText('Get the top 10 users by revenue for the current month')
    ).toBeInTheDocument()
  })

  it('displays folder badge when folder is set', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    expect(screen.getByText('Revenue Reports')).toBeInTheDocument()
  })

  it('displays all tags', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    expect(screen.getByText('analytics')).toBeInTheDocument()
    expect(screen.getByText('users')).toBeInTheDocument()
  })

  it('shows filled star icon when query is favorite', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    const starButton = screen.getByLabelText('Remove from favorites')
    expect(starButton).toBeInTheDocument()
  })

  it('shows sync status when showSyncStatus is true', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} showSyncStatus />)

    expect(screen.getByText('Synced')).toBeInTheDocument()
  })

  it('calls onLoad when card is clicked', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    const card = screen.getByRole('button', { name: /Load query: Top Users Query/i })
    fireEvent.click(card)

    expect(mockHandlers.onLoad).toHaveBeenCalledWith(mockQuery)
  })

  it('calls onToggleFavorite when star is clicked', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    const starButton = screen.getByLabelText('Remove from favorites')
    fireEvent.click(starButton)

    expect(mockHandlers.onToggleFavorite).toHaveBeenCalledWith('test-query-1')
  })

  it('truncates long descriptions', () => {
    const longDescriptionQuery = {
      ...mockQuery,
      description:
        'This is a very long description that should be truncated after 120 characters to ensure the card displays properly without taking up too much vertical space in the list view',
    }

    render(<QueryCard query={longDescriptionQuery} {...mockHandlers} />)

    const description = screen.getByText(/This is a very long description/)
    expect(description.textContent).toContain('...')
  })

  it('shows delete confirmation dialog when delete is clicked', async () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    // Open dropdown menu
    const menuButton = screen.getByLabelText('Query actions')
    fireEvent.click(menuButton)

    // Click delete
    const deleteButton = screen.getByText('Delete')
    fireEvent.click(deleteButton)

    // Confirm dialog is shown
    expect(
      screen.getByText('Are you sure you want to delete "Top Users Query"?')
    ).toBeInTheDocument()
  })

  it('supports keyboard navigation (Enter key)', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    const card = screen.getByRole('button', { name: /Load query: Top Users Query/i })
    fireEvent.keyDown(card, { key: 'Enter' })

    expect(mockHandlers.onLoad).toHaveBeenCalledWith(mockQuery)
  })

  it('supports keyboard navigation (Space key)', () => {
    render(<QueryCard query={mockQuery} {...mockHandlers} />)

    const card = screen.getByRole('button', { name: /Load query: Top Users Query/i })
    fireEvent.keyDown(card, { key: ' ' })

    expect(mockHandlers.onLoad).toHaveBeenCalledWith(mockQuery)
  })
})
