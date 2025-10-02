import { useState, useCallback } from 'react'
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
// Native dropdown implementation - no Radix to avoid body scroll lock
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
  const [loading, _setLoading] = useState(false)
  const [searchValue, setSearchValue] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [newClubName, setNewClubName] = useState('')
  const [isCreating, setIsCreating] = useState(false)

  const debouncedSearchValue = useDebounce(searchValue, 300)

  // Find the selected club
  const selectedClub = clubs.find(club => club.id === selectedClubId)

  // Filter clubs based on search
  const filteredClubs = clubs.filter(club =>
    club.name.toLowerCase().includes(debouncedSearchValue.toLowerCase())
  )

  const handleSelectClub = useCallback((club: Club) => {
    onClubSelected(club)
    setOpen(false)
    setSearchValue('')
  }, [onClubSelected])

  const openCreateDialog = () => {
    setNewClubName(searchValue)
    setShowCreateDialog(true)
    setOpen(false)
  }

  const handleCreateClub = useCallback(async () => {
    if (!newClubName.trim()) return

    setIsCreating(true)
    try {
      const createRequest: CreateClubRequest = {
        name: newClubName.trim(),
      }

      const newClub = await apiClient.createClub(createRequest)

      toast.success(t('clubs.createSuccess', `Club "${newClub.name}" created successfully.`))
      setShowCreateDialog(false)
      setNewClubName('')
      setSearchValue('')
      
      // Select the newly created club
      onClubSelected(newClub)
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('clubs.createError', 'Failed to create club.'))
    } finally {
      setIsCreating(false)
    }
  }, [newClubName, onClubSelected, t])

  return (
    <>
      <div className="relative">
        {/* Trigger Button */}
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          onClick={() => setOpen(!open)}
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
                    placeholder={t('clubs.searchClubs')}
                    value={searchValue}
                    onChange={(e) => setSearchValue(e.target.value)}
                  />
                </div>
                
                {/* Scrollable Content - Native scrolling, no body lock */}
                <div className="max-h-60 overflow-y-auto overscroll-contain p-1" style={{ touchAction: 'pan-y', WebkitOverflowScrolling: 'touch' }}>
                  {loading && (
                    <div className="p-4 text-center">
                      <p className="text-sm text-muted-foreground">
                        {t('common.loading')}
                      </p>
                    </div>
                  )}
                  
                  {!loading && filteredClubs.length === 0 && (
                    <div className="p-4 text-center space-y-3">
                      <p className="text-sm text-muted-foreground">
                        {t('clubs.noClubs')}
                      </p>
                      {searchValue && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={openCreateDialog}
                          className="inline-flex items-center space-x-2"
                        >
                          <Add size={14} />
                          <span>{t('clubs.createNew')} "{searchValue}"</span>
                        </Button>
                      )}
                    </div>
                  )}

                  {filteredClubs.map((club) => (
                    <div
                      key={club.id}
                      onClick={() => handleSelectClub(club)}
                      className="flex items-center space-x-2 px-2 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground rounded-sm"
                    >
                      <Buildings2 size={16} />
                      <span className="flex-1">{club.name}</span>
                      <TickCircle
                        className={cn(
                          'ml-auto h-4 w-4',
                          selectedClub?.id === club.id ? 'opacity-100' : 'opacity-0'
                        )}
                      />
                    </div>
                  ))}

                  {!loading && searchValue && filteredClubs.length > 0 && (
                    <div 
                      onClick={openCreateDialog}
                      className="flex items-center space-x-2 w-full text-muted-foreground px-2 py-2 text-sm cursor-pointer hover:bg-accent hover:text-accent-foreground rounded-sm"
                    >
                      <Add size={16} />
                      <span>
                        {t('clubs.createNew')} "{searchValue}"
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </div>

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
              <Label htmlFor="club-name">{t('clubs.name')}</Label>
              <Input
                id="club-name"
                value={newClubName}
                onChange={(e) => setNewClubName(e.target.value)}
                placeholder={t('clubs.namePlaceholder', 'Enter club name...')}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCreateDialog(false)}
              disabled={isCreating}
            >
              {t('common.cancel')}
            </Button>
            <Button
              onClick={handleCreateClub}
              disabled={!newClubName.trim() || isCreating}
            >
              {isCreating ? t('common.creating') + '...' : t('common.create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}