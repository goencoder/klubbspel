import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { PlayerSelector } from '../PlayerSelector'
import { apiClient } from '@/services/api'
import type { Player, ListPlayersResponse } from '@/types/api'

// Mock the API client
vi.mock('@/services/api', () => ({
  apiClient: {
    listPlayers: vi.fn()
  }
}))

// Mock react-i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key
  })
}))

// Mock the testIds
vi.mock('@/lib/testIds', () => ({
  testIds: {
    playerSelector: {
      trigger: (context: string) => `player-selector-trigger-${context}`,
      popover: (context: string) => `player-selector-popover-${context}`,
      searchInput: (context: string) => `player-selector-search-${context}`,
      optionsList: (context: string) => `player-selector-options-${context}`,
      option: (index: number) => `player-selector-option-${index}`
    }
  }
}))

const mockPlayers: Player[] = [
  {
    id: 'player1',
    displayName: 'Test Player 1',
    firstName: 'Test',
    lastName: 'Player 1',
    email: 'test1@example.com',
    active: true,
    clubMemberships: [],
    isPlatformOwner: false,
    lastLoginAt: null
  },
  {
    id: 'player2', 
    displayName: 'Test Player 2',
    firstName: 'Test',
    lastName: 'Player 2',
    email: 'test2@example.com',
    active: true,
    clubMemberships: [],
    isPlatformOwner: false,
    lastLoginAt: null
  },
  {
    id: 'player3',
    displayName: 'Test Player 3', 
    firstName: 'Test',
    lastName: 'Player 3',
    email: 'test3@example.com',
    active: true,
    clubMemberships: [],
    isPlatformOwner: false,
    lastLoginAt: null
  }
]

describe('PlayerSelector', () => {
  const mockOnPlayerSelected = vi.fn()
  
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should load initial players with club filter when provided', async () => {
    const mockResponse: ListPlayersResponse = {
      items: mockPlayers.slice(0, 2),
      hasNextPage: true,
      hasPreviousPage: false,
      startCursor: 'player1',
      endCursor: 'player2'
    }

    vi.mocked(apiClient.listPlayers).mockResolvedValue(mockResponse)

    render(
      <PlayerSelector 
        onPlayerSelected={mockOnPlayerSelected}
        clubId="test-club-id"
      />
    )

    // Open the dropdown
    const trigger = screen.getByRole('combobox')
    fireEvent.click(trigger)

    await waitFor(() => {
      expect(apiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: undefined,
        clubFilter: ['test-club-id'],
        pageSize: 50
      }, 'player-selector')
    })
  })

  it('should load players without club filter for open series', async () => {
    const mockResponse: ListPlayersResponse = {
      items: mockPlayers.slice(0, 2),
      hasNextPage: true,
      hasPreviousPage: false,
      startCursor: 'player1', 
      endCursor: 'player2'
    }

    vi.mocked(apiClient.listPlayers).mockResolvedValue(mockResponse)

    render(
      <PlayerSelector 
        onPlayerSelected={mockOnPlayerSelected}
        // No clubId provided - should load all players
      />
    )

    // Open the dropdown
    const trigger = screen.getByRole('combobox')
    fireEvent.click(trigger)

    await waitFor(() => {
      expect(apiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: undefined,
        clubFilter: undefined,
        pageSize: 50
      }, 'player-selector')
    })
  })

  it('should handle infinite scroll when scrolling to bottom', async () => {
    const initialResponse: ListPlayersResponse = {
      items: mockPlayers.slice(0, 2),
      hasNextPage: true,
      hasPreviousPage: false,
      startCursor: 'player1',
      endCursor: 'player2'
    }

    const moreResponse: ListPlayersResponse = {
      items: [mockPlayers[2]],
      hasNextPage: false,
      hasPreviousPage: true,
      startCursor: 'player3',
      endCursor: 'player3'
    }

    vi.mocked(apiClient.listPlayers)
      .mockResolvedValueOnce(initialResponse)
      .mockResolvedValueOnce(moreResponse)

    render(
      <PlayerSelector 
        onPlayerSelected={mockOnPlayerSelected}
        clubId="test-club-id"
      />
    )

    // Open the dropdown
    const trigger = screen.getByRole('combobox')
    fireEvent.click(trigger)

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('Test Player 1')).toBeInTheDocument()
    })

    // Find the scrollable container and simulate scroll to bottom
    const scrollContainer = screen.getByRole('group').parentElement
    expect(scrollContainer).toBeInTheDocument()

    // Simulate scrolling to bottom
    Object.defineProperty(scrollContainer!, 'scrollTop', { value: 250, writable: true })
    Object.defineProperty(scrollContainer!, 'scrollHeight', { value: 300, writable: true })  
    Object.defineProperty(scrollContainer!, 'clientHeight', { value: 200, writable: true })

    fireEvent.scroll(scrollContainer!)

    // Should trigger loading more players
    await waitFor(() => {
      expect(apiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: undefined,
        clubFilter: ['test-club-id'],
        pageSize: 25,
        cursorAfter: 'player2'
      }, 'player-selector-more')
    })
  })
})