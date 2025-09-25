import { useState, useEffect, useCallback, forwardRef, useImperativeHandle, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { buttonVariants } from '@/components/ui/button'
// Completely custom implementation without Radix to avoid body scroll lock
import { TickCircle, ArrowSwapVertical } from 'iconsax-reactjs'
import { cn } from '@/lib/utils'
import { useDebounce } from '@/hooks/useDebounce'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient, handleApiError } from '@/services/api'
import type { Player } from '@/types/api'
import { toast } from 'sonner'

interface PlayerSelectorProps {
  value?: string
  onPlayerSelected: (player: Player) => void
  clubId?: string
  excludePlayerId?: string
  className?: string
}

export type PlayerSelectorHandle = {
  focus: () => void
}

export const PlayerSelector = forwardRef<PlayerSelectorHandle, PlayerSelectorProps>(function PlayerSelector({
  value,
  onPlayerSelected,
  clubId,
  excludePlayerId,
  className
}: PlayerSelectorProps, ref) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [players, setPlayers] = useState<Player[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasNextPage, setHasNextPage] = useState(false)
  const [_nextPageToken, setNextPageToken] = useState<string | undefined>()
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null)
  const [clubNames, setClubNames] = useState<Map<string, string>>(new Map())
  const triggerRef = useRef<HTMLButtonElement | null>(null)
  const scrollContainerRef = useRef<HTMLDivElement | null>(null)
  
  // Use refs for values that change frequently to avoid recreating loadPlayers
  const clubNamesRef = useRef<Map<string, string>>(new Map())
  const nextPageTokenRef = useRef<string | undefined>(undefined)

  const debouncedSearchQuery = useDebounce(searchQuery, 300)
  
  // Keep refs in sync with state
  useEffect(() => {
    clubNamesRef.current = clubNames
  }, [clubNames])

  useImperativeHandle(ref, () => ({
    focus: () => {
      triggerRef.current?.focus()
    }
  }))

  const loadPlayers = useCallback(async (isLoadMore = false) => {
    try {
      if (isLoadMore) {
        setLoadingMore(true)
      } else {
        setLoading(true)
      }
      
      const response = await apiClient.listPlayers({
        searchQuery: debouncedSearchQuery || undefined,
        clubId: clubId,
        pageSize: 50,
        cursorAfter: isLoadMore ? nextPageTokenRef.current : undefined
      }, 'player-selector')

      // Filter out excluded player
      const filteredPlayers = response.items.filter(p => 
        p.active && p.id !== excludePlayerId
      )
      
      if (isLoadMore) {
        setPlayers(prev => [...prev, ...filteredPlayers])
      } else {
        setPlayers(filteredPlayers)
      }
      
      setHasNextPage(response.hasNextPage)
      setNextPageToken(response.endCursor)
      nextPageTokenRef.current = response.endCursor

      // Fetch club names for clubs we don't have cached - simplified approach
      const clubIdsToFetch = new Set<string>()
      filteredPlayers.forEach(player => {
        player.clubMemberships?.forEach(membership => {
          if (membership.active && !clubNamesRef.current.has(membership.clubId)) {
            clubIdsToFetch.add(membership.clubId)
          }
        })
      })

      // Fetch missing club names asynchronously without affecting search state
      if (clubIdsToFetch.size > 0) {
        Promise.allSettled(
          Array.from(clubIdsToFetch).map(async (clubId) => {
            try {
              const club = await apiClient.getClub(clubId)
              setClubNames(prev => {
                const newMap = new Map(prev)
                if (!newMap.has(clubId)) {
                  newMap.set(clubId, club.name)
                  clubNamesRef.current.set(clubId, club.name)
                }
                return newMap
              })
            } catch (_error) {
              setClubNames(prev => {
                const newMap = new Map(prev)
                if (!newMap.has(clubId)) {
                  newMap.set(clubId, t('common.unknownClub', 'Unknown Club'))
                  clubNamesRef.current.set(clubId, t('common.unknownClub', 'Unknown Club'))
                }
                return newMap
              })
            }
          })
        )
      }
    } catch (error: unknown) {
      handleApiError(error, (apiError) => {
        toast.error(apiError.message || t('errors.generic'))
      })
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }, [debouncedSearchQuery, clubId, excludePlayerId, t])

  const loadMorePlayers = useCallback(() => {
    if (hasNextPage && !loadingMore) {
      loadPlayers(true)
    }
  }, [hasNextPage, loadingMore, loadPlayers])

  // Add throttling for scroll events
  const throttleRef = useRef<number | null>(null)
  
  // Handle scroll for infinite loading with throttling
  const handleScroll = useCallback((event: React.UIEvent<HTMLDivElement>) => {
    if (throttleRef.current) return // Skip if already throttled
    
    throttleRef.current = window.setTimeout(() => {
      throttleRef.current = null
      
      const target = event.target as HTMLDivElement
      const { scrollTop, scrollHeight, clientHeight } = target
      
      // Trigger load more when scrolled to within 100px of bottom
      const threshold = 100
      const isNearBottom = scrollHeight - scrollTop - clientHeight < threshold
      
      // Only trigger if we're near bottom, have more pages, and not already loading
      if (isNearBottom && hasNextPage && !loadingMore && !loading) {
        loadMorePlayers()
      }
    }, 150) // Throttle to max 150ms
  }, [hasNextPage, loadingMore, loading, loadMorePlayers])

  useEffect(() => {
    // Reset pagination when search query changes
    setNextPageToken(undefined)
    nextPageTokenRef.current = undefined
    setHasNextPage(false)
    
    if (open || debouncedSearchQuery) {
      loadPlayers()
    }
  }, [open, debouncedSearchQuery, loadPlayers])

  useEffect(() => {
    if (!value) {
      if (selectedPlayer) {
        setSelectedPlayer(null)
      }
      return
    }

    if (selectedPlayer?.id === value) {
      return
    }

    const player = players.find(p => p.id === value)
    if (player) {
      setSelectedPlayer(player)
    }
  }, [value, players, selectedPlayer])

  const handlePlayerSelect = (player: Player) => {
    setSelectedPlayer(player)
    setOpen(false)
    onPlayerSelected(player)
  }

  return (
    <div className="relative">
      {/* Trigger Button */}
      <button
        ref={triggerRef}
        type="button"
        role="combobox"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={cn(buttonVariants({ variant: 'outline' }), 'w-full justify-between text-left', className)}
      >
        {selectedPlayer ? (
          <div className="flex-1 min-w-0">
            <div className="font-medium truncate">{selectedPlayer.displayName}</div>
            {(() => {
              const primaryClub = selectedPlayer.clubMemberships?.find(m => m.active) || selectedPlayer.clubMemberships?.[0]
              const clubName = primaryClub ? clubNames.get(primaryClub.clubId) : undefined
              return clubName ? (
                <div className="text-xs text-muted-foreground truncate">{clubName}</div>
              ) : null
            })()}
          </div>
        ) : (
          <span className="text-muted-foreground">{t('players.selectPlayer')}</span>
        )}
        <ArrowSwapVertical size={16} className="ml-2 opacity-50 flex-shrink-0" />
      </button>

      {/* Native Dropdown - No Radix, No Body Scroll Lock */}
      {open && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 z-40" 
            onClick={() => setOpen(false)}
          />
          
          {/* Dropdown Content */}
          <div className="absolute top-full left-0 right-0 z-50 mt-1 bg-popover text-popover-foreground rounded-md border shadow-md">
            <div className="flex flex-col">
              {/* Search Input */}
              <div className="flex items-center border-b px-3">
                <svg
                  className="mr-2 h-4 w-4 shrink-0 opacity-50"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <circle cx="11" cy="11" r="8" />
                  <path d="m21 21-4.35-4.35" />
                </svg>
                <input
                  autoFocus
                  className="flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
                  placeholder={t('common.search') + '...'}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              
              {/* Scrollable Content - Now truly native scrolling */}
              <div 
                ref={scrollContainerRef}
                className="max-h-60 overflow-y-auto overscroll-contain" 
                onScroll={handleScroll}
                style={{ 
                  scrollbarGutter: 'stable',
                  // Force scrolling behavior
                  touchAction: 'pan-y',
                  WebkitOverflowScrolling: 'touch'
                }}
              >
                {loading && players.length === 0 && (
                  <div className="py-4 text-center">
                    <LoadingSpinner size="sm" />
                  </div>
                )}
                
                {!loading && players.length === 0 && (
                  <div className="py-4 text-center">
                    <p className="text-sm text-muted-foreground">
                      {t('players.noPlayers')}
                    </p>
                  </div>
                )}
                
                {players.map((player) => {
                  // Get the primary active club membership for display
                  const primaryClub = player.clubMemberships?.find(m => m.active) || player.clubMemberships?.[0]
                  const clubName = primaryClub ? clubNames.get(primaryClub.clubId) : undefined
                  
                  return (
                    <div
                      key={player.id}
                      onClick={() => handlePlayerSelect(player)}
                      className="flex items-center px-2 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground rounded-sm"
                    >
                      <TickCircle
                        size={16}
                        className={cn(
                          "mr-2 flex-shrink-0",
                          selectedPlayer?.id === player.id ? "opacity-100" : "opacity-0"
                        )}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium truncate">{player.displayName}</div>
                        {clubName && (
                          <div className="text-xs text-muted-foreground mt-0.5 truncate">
                            {clubName}
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
                
                {hasNextPage && (
                  <div className="border-t border-border bg-background">
                    <button
                      onClick={loadMorePlayers}
                      disabled={loadingMore}
                      className="w-full py-3 px-4 text-sm font-medium text-primary hover:bg-accent hover:text-accent-foreground transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                    >
                      {loadingMore ? (
                        <>
                          <LoadingSpinner size="sm" />
                          {t('common.loading')}
                        </>
                      ) : (
                        <>
                          <span>{t('common.loadMore', 'Load more...')}</span>
                          <span className="text-xs opacity-70">⬇</span>
                        </>
                      )}
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
})
