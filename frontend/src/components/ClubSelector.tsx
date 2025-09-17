import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { TickCircle, ArrowDown2, Add, Buildings2 } from 'iconsax-reactjs'
import { cn } from '@/lib/utils'
import { apiClient } from '@/services/api'
import type { Club, CreateClubRequest, ApiError } from '@/types/api'
import { useDebounce } from '@/hooks/useDebounce'

interface ClubSelectorProps {
  clubs: Club[]
  selectedClubId?: string
  onClubSelected: (club: Club | null) => void
  disabled?: boolean
  placeholder?: string
  required?: boolean
}

export function ClubSelector({
  clubs,
  selectedClubId,
  onClubSelected,
  disabled,
  placeholder = 'Select club...',
}: ClubSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [searchValue, setSearchValue] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [newClubName, setNewClubName] = useState('')

  const debouncedSearch = useDebounce(searchValue, 300)

  const selectedClub = clubs.find((club) => club.id === selectedClubId) || null

  const loadClubs = useCallback(
    async (_search?: string) => {
      try {
        setLoading(true)
        // Note: This component now receives clubs as props, so we don't need to load them here
        // This function is kept for potential future search functionality
      } catch (error) {
        const apiError = error as ApiError
        toast.error(apiError?.message || t('errors.unexpectedError'))
      } finally {
        setLoading(false)
      }
    },
    [t]
  )

  // Load clubs on mount and whenever search changes
  useEffect(() => {
    loadClubs(debouncedSearch)
  }, [loadClubs, debouncedSearch])

  const handleCreateClub = async () => {
    if (!newClubName.trim()) {
      toast.error(t('clubs.validation.nameRequired'))
      return
    }
    if (newClubName.trim().length < 2) {
      toast.error(t('clubs.validation.nameMinLength'))
      return
    }
    if (newClubName.trim().length > 80) {
      toast.error(t('clubs.validation.nameMaxLength'))
      return
    }

    try {
      setSubmitting(true)
      const clubData: CreateClubRequest = { name: newClubName.trim() }
      const club = await apiClient.createClub(clubData)

      // Add to list and select it
      // setClubs((prev) => [club, ...prev])
      onClubSelected(club)

      // Reset and close
      setShowCreateDialog(false)
      setNewClubName('')
      setOpen(false)

      toast.success(t('clubs.created'))
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError?.message || t('errors.unexpectedError'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleSelectClub = (club: Club) => {
    onClubSelected(club)
    setOpen(false)
  }

  const openCreateDialog = () => {
    setNewClubName(searchValue)
    setShowCreateDialog(true)
    setOpen(false)
  }

  return (
    <>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn('w-full justify-between', !selectedClub && 'text-muted-foreground')}
            disabled={disabled}
          >
            <div className="flex items-center space-x-2">
              <Buildings2 size={16} />
              <span className="truncate">
                {selectedClub ? selectedClub.name : (placeholder || t('clubs.selectClub'))}
              </span>
            </div>
            <ArrowDown2 size={16} className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-full p-0" align="start">
          <Command>
            <CommandInput
              placeholder={t('clubs.searchClubs')}
              value={searchValue}
              onValueChange={setSearchValue}
            />
            <CommandList>
              <CommandEmpty>
                <div className="p-4 text-center space-y-3">
                  <p className="text-sm text-muted-foreground">
                    {loading ? t('common.loading') : t('clubs.noClubs')}
                  </p>
                  {!loading && searchValue && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={openCreateDialog}
                      className="inline-flex items-center space-x-2"
                    >
                      <Add size={14} />
                      <span>{t('clubs.createNew')} “{searchValue}”</span>
                    </Button>
                  )}
                </div>
              </CommandEmpty>

              <CommandGroup>
                {clubs.map((club) => (
                  <CommandItem
                    key={club.id}
                    value={club.name}
                    onSelect={() => handleSelectClub(club)}
                    className="flex items-center space-x-2"
                  >
                    <Buildings2 size={16} />
                    <span className="flex-1">{club.name}</span>
                    <TickCircle
                      className={cn(
                        'ml-auto h-4 w-4',
                        selectedClub?.id === club.id ? 'opacity-100' : 'opacity-0'
                      )}
                    />
                  </CommandItem>
                ))}

                {!loading && searchValue && clubs.length > 0 && (
                  <CommandItem onSelect={openCreateDialog}>
                    <div className="flex items-center space-x-2 w-full text-muted-foreground">
                      <Add size={16} />
                      <span>
                        {t('clubs.createNew')} “{searchValue}”
                      </span>
                    </div>
                  </CommandItem>
                )}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>

      {/* Create Club Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('clubs.createNew')}</DialogTitle>
            <DialogDescription>
              {t('clubs.createHelp', 'Create a new table tennis club.')}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="new-club-name">{t('clubs.name')} *</Label>
              <Input
                id="new-club-name"
                value={newClubName}
                onChange={(e) => setNewClubName(e.target.value)}
                placeholder={t('clubs.name')}
                maxLength={80}
                autoFocus
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={handleCreateClub} disabled={submitting || !newClubName.trim()}>
              {submitting ? t('common.loading') : t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
