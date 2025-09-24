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
import { testIds } from '@/lib/testIds'

interface PlayerSelectorProps {
  value?: string
  onPlayerSelected: (player: Player) => void
  clubId?: string
  excludePlayerId?: string
  className?: string
  id?: string
}

export type PlayerSelectorHandle = {
  focus: () => void
}

export const PlayerSelector = forwardRef<PlayerSelectorHandle, PlayerSelectorProps>(function PlayerSelector({
  value,
  onPlayerSelected,
  clubId,
  excludePlayerId,
  className,
  id
}: PlayerSelectorProps, ref) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [players, setPlayers] = useState<Player[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null)
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

  // Generate IDs based on context
  const selectorContext = id || 'default'
  const triggerId = testIds.playerSelector.trigger(selectorContext)
  const popoverId = testIds.playerSelector.popover(selectorContext)
  const searchId = testIds.playerSelector.searchInput(selectorContext)
  const optionsId = testIds.playerSelector.optionsList(selectorContext)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          id={triggerId}
          ref={triggerRef}
          type="button"
          role="combobox"
          aria-expanded={open}
          className={cn(buttonVariants({ variant: 'outline' }), 'w-full justify-between', className)}
        >
          {selectedPlayer ? selectedPlayer.displayName : t('players.selectPlayer')}
          <ArrowSwapVertical size={16} className="ml-2 opacity-50" />
        </button>
      </PopoverTrigger>
      <PopoverContent id={popoverId} className="w-full p-0" align="start">
        <Command>
          <CommandInput 
            id={searchId}
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
          <CommandGroup id={optionsId}>
            {loading && (
              <div className="py-2">
                <LoadingSpinner size="sm" />
              </div>
            )}
            {players.map((player, index) => (
              <CommandItem
                key={player.id}
                id={testIds.playerSelector.option(index)}
                onSelect={() => handlePlayerSelect(player)}
                className="cursor-pointer"
                data-player-id={player.id}
                data-player-name={player.displayName}
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
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  )
})
