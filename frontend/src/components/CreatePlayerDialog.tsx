import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ClubSelector } from '@/components/ClubSelector'
import { PlayerConfirmDialog } from '@/components/PlayerConfirmDialog'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { Player, Club } from '@/types/api'
import { toast } from 'sonner'

interface CreatePlayerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onPlayerCreated: (player: Player) => void
}

export function CreatePlayerDialog({ open, onOpenChange, onPlayerCreated }: CreatePlayerDialogProps) {
  const { t } = useTranslation()
  const { isPlatformOwner, isClubAdmin, selectedClubId } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [clubs, setClubs] = useState<Club[]>([])
  const [formData, setFormData] = useState({
    displayName: '',
    clubId: ''
  })
  const [similarPlayers, setSimilarPlayers] = useState<Player[]>([])
  const [showSimilarDialog, setShowSimilarDialog] = useState(false)
  const [hasManualClubSelection, setHasManualClubSelection] = useState(false)

  const loadManageableClubs = useCallback(async () => {
    try {
      const response = await apiClient.listClubs({ pageSize: 100 })

      // Filter clubs based on user permissions
      const manageableClubs: Club[] = []

      // Check if user is platform owner first
      const userIsPlatformOwner = isPlatformOwner()

      for (const club of response.items) {
        // Platform owners can manage any club
        if (userIsPlatformOwner) {
          manageableClubs.push(club)
          continue
        }

        // Regular users can only manage clubs they are admin of
        const isAdmin = isClubAdmin(club.id)
        if (isAdmin) {
          manageableClubs.push(club)
        }
      }

      setClubs(manageableClubs)

      if (!hasManualClubSelection) {
        setFormData(prev => {
          const selectedClubExists = selectedClubId
            ? manageableClubs.some(club => club.id === selectedClubId)
            : false

          if (selectedClubExists) {
            return prev.clubId === selectedClubId
              ? prev
              : { ...prev, clubId: selectedClubId }
          }

          if (manageableClubs.length === 1) {
            const [onlyClub] = manageableClubs
            return prev.clubId === onlyClub.id
              ? prev
              : { ...prev, clubId: onlyClub.id }
          }

          if (prev.clubId && !manageableClubs.some(club => club.id === prev.clubId)) {
            return { ...prev, clubId: '' }
          }

          return prev
        })
      }

      // Show warning if no manageable clubs - handled silently
    } catch (error: unknown) {
      toast.error((error as Error).message || t('errors.generic'))
    }
  }, [hasManualClubSelection, isPlatformOwner, isClubAdmin, selectedClubId, t])

  // Load clubs when dialog opens
  useEffect(() => {
    if (open) {
      loadManageableClubs()
    } else {
      // Reset form when dialog closes
      setFormData({
        displayName: '',
        clubId: ''
      })
      setSimilarPlayers([])
      setShowSimilarDialog(false)
      setClubs([])
      setHasManualClubSelection(false)
    }
  }, [open, loadManageableClubs])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!formData.displayName.trim()) {
      toast.error(t('players.validation.nameRequired'))
      return
    }

    if (!formData.clubId) {
      toast.error(t('players.validation.clubRequired'))
      return
    }

    await createPlayer()
  }

  const createPlayer = async () => {
    try {
      setLoading(true)
      
      const createRequest = {
        displayName: formData.displayName.trim(),
        initialClubId: formData.clubId
      }

      const response = await apiClient.createPlayer(createRequest)
      
      // Check if there are similar players
      if (response.similar && response.similar.length > 0) {
        setSimilarPlayers(response.similar)
        setShowSimilarDialog(true)
        setLoading(false)
        return
      }

      // No similar players, proceed with creation
      onPlayerCreated(response.player)
      onOpenChange(false)
      toast.success(t('players.created'))
    } catch (error: unknown) {
      const errorMessage = (error as Error).message || '';
      // Handle specific authorization errors
      if (errorMessage.includes('CLUB_ADMIN_OR_PLATFORM_OWNER_REQUIRED')) {
        toast.error(t('errors.clubAdminRequired'))
      } else if (errorMessage.includes('CLUB_ID_REQUIRED_FOR_NON_PLATFORM_OWNERS')) {
        toast.error(t('errors.clubIdRequired'))
      } else if (errorMessage.includes('LOGIN_REQUIRED')) {
        toast.error(t('auth.loginRequired'))
      } else {
        toast.error(errorMessage || t('errors.generic'))
      }
    } finally {
      setLoading(false)
    }
  }

  const handleClubSelected = (club: Club | null) => {
    setHasManualClubSelection(true)
    setFormData(prev => ({ ...prev, clubId: club?.id || '' }))
  }

  const handleUseSimilarPlayer = (player: Player) => {
    onPlayerCreated(player)
    setShowSimilarDialog(false)
    onOpenChange(false)
    toast.success(t('common.success'))
  }

  const handleCreateNewAnyway = async () => {
    setShowSimilarDialog(false)
    // Continue with the creation process
    await createPlayer()
  }

  const canCreatePlayers = isPlatformOwner() || clubs.length > 0

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('players.create')}</DialogTitle>
            <DialogDescription>
              {canCreatePlayers
                ? t('players.createDescription')
                : t('players.noManageableClubs')
              }
            </DialogDescription>
          </DialogHeader>

          {!canCreatePlayers ? (
            <div className="py-6 text-center text-muted-foreground">
              <p>{t('players.needClubAdmin')}</p>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="player-name" className="flex items-center gap-1">
                  {t('players.name')}
                  <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="player-name"
                  type="text"
                  placeholder={t('players.namePlaceholder')}
                  value={formData.displayName}
                  onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
                  className={!formData.displayName.trim() ? 'border-red-300 focus:border-red-500' : ''}
                  disabled={loading}
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="club-select" className="flex items-center gap-1">
                  {t('players.club')}
                  <span className="text-red-500">*</span>
                </Label>
                <ClubSelector
                  clubs={clubs}
                  selectedClubId={formData.clubId}
                  onClubSelected={handleClubSelected}
                  placeholder={t('players.selectClub')}
                  disabled={loading}
                />
              </div>

              <DialogFooter>
                <Button
                  type="submit"
                  disabled={loading || !formData.displayName.trim() || !formData.clubId}
                  className="min-w-[100px]"
                >
                  {loading ? <LoadingSpinner size="sm" /> : t('players.create')}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <PlayerConfirmDialog
        open={showSimilarDialog}
        onOpenChange={setShowSimilarDialog}
        playerName={formData.displayName}
        similarPlayers={similarPlayers}
        onUseSimilarPlayer={handleUseSimilarPlayer}
        onCreateNewAnyway={handleCreateNewAnyway}
      />
    </>
  )
}