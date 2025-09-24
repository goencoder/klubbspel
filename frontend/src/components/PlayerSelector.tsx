import { useState, useEffect, useCallback, forwardRef, useImperativeHandle, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { buttonVariants } from '@/components/ui/button'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem } from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { TickCircle, ArrowSwapVertical } from 'iconsax-reactjs'
import { cn } from '@/lib/utils'
import { useDebounce } from '@/hooks/useDebounce'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
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
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null)
  const [clubNames, setClubNames] = useState<Map<string, string>>(new Map())
  const triggerRef = useRef<HTMLButtonElement | null>(null)

  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  useImperativeHandle(ref, () => ({
    focus: () => {
      triggerRef.current?.focus()
    }
  }))

  const loadPlayers = useCallback(async () => {
    try {
      setLoading(true)
      const response = await apiClient.listPlayers({
        searchQuery: debouncedSearchQuery || undefined,
        clubId: clubId,
        pageSize: 50
      }, 'player-selector')

      // Filter out excluded player
      const filteredPlayers = response.items.filter(p => 
        p.active && p.id !== excludePlayerId
      )
      
      setPlayers(filteredPlayers)

            // Fetch club names for clubs we don't have cached - simplified approach
      const clubIdsToFetch = new Set<string>()
      filteredPlayers.forEach(player => {
        player.clubMemberships?.forEach(membership => {
          if (membership.active) {
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
                if (!newMap.has(clubId)) { // Only set if not already cached
                  newMap.set(clubId, club.name)
                }
                return newMap
              })
            } catch (_error) {
              setClubNames(prev => {
                const newMap = new Map(prev)
                if (!newMap.has(clubId)) { // Only set if not already cached
                  newMap.set(clubId, t('common.unknownClub', 'Unknown Club'))
                }
                return newMap
              })
            }
          })
        )
      }
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.generic'))
    } finally {
      setLoading(false)
    }
  }, [debouncedSearchQuery, clubId, excludePlayerId, t])

  useEffect(() => {
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
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          ref={triggerRef}
          type="button"
          role="combobox"
          aria-expanded={open}
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
      </PopoverTrigger>
      <PopoverContent className="w-full p-0" align="start">
        <Command shouldFilter={false}>
          <CommandInput 
            placeholder={t('common.search') + '...'}
            value={searchQuery}
            onValueChange={setSearchQuery}
          />
          <CommandEmpty>
            <div className="py-4 text-center">
              <p className="text-sm text-muted-foreground">
                {loading ? t('common.loading') : t('players.noPlayers')}
              </p>
            </div>
          </CommandEmpty>
          <CommandGroup>
            {loading && (
              <div className="py-2">
                <LoadingSpinner size="sm" />
              </div>
            )}
            {players.map((player) => {
              // Get the primary active club membership for display
              const primaryClub = player.clubMemberships?.find(m => m.active) || player.clubMemberships?.[0]
              const clubName = primaryClub ? clubNames.get(primaryClub.clubId) : undefined
              
              return (
                <CommandItem
                  key={player.id}
                  onSelect={() => handlePlayerSelect(player)}
                  className="cursor-pointer"
                >
                  <TickCircle
                    size={16}
                    className={cn(
                      "mr-2",
                      selectedPlayer?.id === player.id ? "opacity-100" : "opacity-0"
                    )}
                  />
                  <div className="flex-1">
                    <div className="font-medium">{player.displayName}</div>
                    {clubName && (
                      <div className="text-xs text-muted-foreground mt-0.5">
                        {clubName}
                      </div>
                    )}
                  </div>
                </CommandItem>
              )
            })}
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  )
})
