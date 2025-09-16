import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
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

export function PlayerSelector({ 
  value, 
  onPlayerSelected, 
  clubId, 
  excludePlayerId,
  className 
}: PlayerSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [players, setPlayers] = useState<Player[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedPlayer, setSelectedPlayer] = useState<Player | null>(null)

  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  useEffect(() => {
    if (open || debouncedSearchQuery) {
      loadPlayers()
    }
  }, [debouncedSearchQuery, open, clubId, excludePlayerId])

  useEffect(() => {
    if (value && !selectedPlayer) {
      // Find the selected player by ID
      const player = players.find(p => p.id === value)
      if (player) {
        setSelectedPlayer(player)
      }
    }
  }, [value, players, selectedPlayer])

  const loadPlayers = async () => {
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
    } catch (error: any) {
      toast.error(error.message || t('errors.generic'))
    } finally {
      setLoading(false)
    }
  }

  const handlePlayerSelect = (player: Player) => {
    setSelectedPlayer(player)
    setOpen(false)
    onPlayerSelected(player)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn("w-full justify-between", className)}
        >
          {selectedPlayer ? selectedPlayer.displayName : t('players.selectPlayer')}
          <ArrowSwapVertical size={16} className="ml-2 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-full p-0" align="start">
        <Command>
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
            {players.map((player) => (
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
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  )
}