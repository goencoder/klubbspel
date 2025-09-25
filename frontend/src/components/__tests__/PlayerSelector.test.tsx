/**
 * @jest-environment jsdom
 */
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { PlayerSelector } from '../PlayerSelector'
import { apiClient } from '@/services/api'
import type { Player } from '@/types/api'

// Mock the dependencies
jest.mock('@/services/api')
jest.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback || key
  })
}))
jest.mock('sonner', () => ({
  toast: {
    error: jest.fn()
  }
}))

const mockApiClient = apiClient as jest.Mocked<typeof apiClient>

const mockPlayers: Player[] = [
  {
    id: '1',
    displayName: 'Thomas Eriksson',
    active: true,
    email: 'thomas@example.com',
    clubMemberships: [{
      clubId: 'club1',
      role: 'MEMBERSHIP_ROLE_MEMBER',
      active: true,
      joinedAt: new Date().toISOString()
    }]
  },
  {
    id: '2', 
    displayName: 'Tomas Ericsson',
    active: true,
    email: 'tomas@example.com',
    clubMemberships: [{
      clubId: 'club1',
      role: 'MEMBERSHIP_ROLE_MEMBER',
      active: true,
      joinedAt: new Date().toISOString()
    }]
  }
]

describe('PlayerSelector', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    
    mockApiClient.listPlayers.mockResolvedValue({
      items: mockPlayers,
      hasNextPage: false,
      hasPreviousPage: false,
      startCursor: '',
      endCursor: ''
    })
    
    mockApiClient.getClub.mockResolvedValue({
      id: 'club1',
      name: 'Test Club',
      supportedSports: []
    })
  })

  test('renders with placeholder text', () => {
    const mockOnSelect = jest.fn()
    
    render(
      <PlayerSelector
        onPlayerSelected={mockOnSelect}
        clubId="club1"
      />
    )
    
    expect(screen.getByText('players.selectPlayer')).toBeInTheDocument()
  })

  test('loads players when opened', async () => {
    const mockOnSelect = jest.fn()
    
    render(
      <PlayerSelector
        onPlayerSelected={mockOnSelect}
        clubId="club1"
      />
    )
    
    // Click to open dropdown
    fireEvent.click(screen.getByRole('combobox'))
    
    await waitFor(() => {
      expect(mockApiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: undefined,
        clubId: 'club1',
        pageSize: 50,
        cursorAfter: undefined
      }, 'player-selector')
    })
  })

  test('filters players based on search query', async () => {
    const mockOnSelect = jest.fn()
    
    render(
      <PlayerSelector
        onPlayerSelected={mockOnSelect}
        clubId="club1"
      />
    )
    
    // Click to open dropdown
    fireEvent.click(screen.getByRole('combobox'))
    
    // Wait for initial load
    await waitFor(() => {
      expect(mockApiClient.listPlayers).toHaveBeenCalled()
    })
    
    // Type in search
    const searchInput = screen.getByPlaceholderText('common.search...')
    fireEvent.change(searchInput, { target: { value: 'Thomas' } })
    
    // Wait for debounced search
    await waitFor(() => {
      expect(mockApiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: 'Thomas',
        clubId: 'club1',
        pageSize: 50,
        cursorAfter: undefined
      }, 'player-selector')
    }, { timeout: 1000 })
  })

  test('handles load more functionality', async () => {
    // Mock response with pagination
    mockApiClient.listPlayers.mockResolvedValueOnce({
      items: [mockPlayers[0]],
      hasNextPage: true,
      hasPreviousPage: false,
      startCursor: '',
      endCursor: 'cursor1'
    }).mockResolvedValueOnce({
      items: [mockPlayers[1]], 
      hasNextPage: false,
      hasPreviousPage: true,
      startCursor: 'cursor1',
      endCursor: 'cursor2'
    })
    
    const mockOnSelect = jest.fn()
    
    render(
      <PlayerSelector
        onPlayerSelected={mockOnSelect}
        clubId="club1"
      />
    )
    
    // Click to open dropdown
    fireEvent.click(screen.getByRole('combobox'))
    
    await waitFor(() => {
      expect(screen.getByText('Thomas Eriksson')).toBeInTheDocument()
    })
    
    // Click load more button
    const loadMoreButton = screen.getByText('common.loadMore')
    fireEvent.click(loadMoreButton)
    
    await waitFor(() => {
      expect(mockApiClient.listPlayers).toHaveBeenCalledWith({
        searchQuery: undefined,
        clubId: 'club1',
        pageSize: 50,
        cursorAfter: 'cursor1'
      }, 'player-selector')
    })
    
    await waitFor(() => {
      expect(screen.getByText('Tomas Ericsson')).toBeInTheDocument()
    })
  })

  test('selects a player', async () => {
    const mockOnSelect = jest.fn()
    
    render(
      <PlayerSelector
        onPlayerSelected={mockOnSelect}
        clubId="club1"
      />
    )
    
    // Click to open dropdown
    fireEvent.click(screen.getByRole('combobox'))
    
    await waitFor(() => {
      expect(screen.getByText('Thomas Eriksson')).toBeInTheDocument()
    })
    
    // Click on a player
    fireEvent.click(screen.getByText('Thomas Eriksson'))
    
    expect(mockOnSelect).toHaveBeenCalledWith(mockPlayers[0])
  })
})