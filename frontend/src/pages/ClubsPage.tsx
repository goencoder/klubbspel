import { ProfileCompletionModal } from '@/components/ProfileCompletionModal'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useDebounce } from '@/hooks/useDebounce'
import { apiClient, handleApiError } from '@/services/api'
import { useAuthStore } from '@/store/auth'
import type { ApiError, Club, CreateClubRequest, Player, UpdateClubRequest } from '@/types/api'
import { Add, Buildings2, Edit2, Eye, SearchNormal1, Trash, User } from 'iconsax-reactjs'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { PageWrapper, PageHeaderSection, HeaderContent, SearchSection, LoadingGrid, SharedEmptyState, ActionGroup } from './Styles'
import { DEFAULT_SPORT, sportIconComponent, sportTranslationKey } from '@/lib/sports'

export function ClubsPage() {
  const { t } = useTranslation()
  // Use global club selection from auth store
  const { user, refreshUser, refreshUserMemberships, selectedClubId } = useAuthStore()
  const [clubs, setClubs] = useState<Club[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [editingClub, setEditingClub] = useState<Club | null>(null)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showProfileModal, setShowProfileModal] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [createFormData, setCreateFormData] = useState<CreateClubRequest>({
    name: ''
  })
  const [editFormData, setEditFormData] = useState<CreateClubRequest>({
    name: ''
  })
  const [selectedClub, setSelectedClub] = useState<Club | null>(null)
  const [showClubDetails, setShowClubDetails] = useState(false)
  const [clubPlayers, setClubPlayers] = useState<Record<string, { count: number; players: Player[] }>>({})
  const [loadingClubPlayers, setLoadingClubPlayers] = useState<Record<string, boolean>>({})

  const debouncedSearch = useDebounce(searchQuery, 300)

  // Check if user's profile is complete (has first and last name)
  const isProfileComplete = user && user.displayName && user.displayName !== user.email && user.displayName.includes(' ')

  const handleCreateClubClick = () => {
    if (!user) {
      toast.error(t('auth.loginRequired'))
      return
    }

    if (!isProfileComplete) {
      setShowProfileModal(true)
      return
    }

    openCreateDialog()
  }

  const handleProfileUpdated = async () => {
    // Refresh user data to get updated profile
    await refreshUser()
    // Now open the create club dialog
    openCreateDialog()
  }

  const loadClubs = useCallback(async (search?: string) => {
    try {
      setLoading(true)
      const response = await apiClient.listClubs(
          { searchQuery: search, pageSize: 50 },
          'clubs-list'
      )

      // Filter clubs based on global club selection
      let filteredClubs = response.items

      if (selectedClubId === 'my-clubs') {
        // Show only clubs where user is a member (including admins)
        const userClubIds = user?.memberships?.map(m => m.clubId) || []
        filteredClubs = response.items.filter(club => {
          return userClubIds.includes(club.id)
        })
      } else if (selectedClubId) {
        // Show specific club
        filteredClubs = response.items.filter(club => {
          return club.id === selectedClubId
        })
      }
      // If selectedClubId is null (All Clubs), show all clubs

      setClubs(filteredClubs)
    } catch (error) {
      handleApiError(error, (apiError) => {
        toast.error(apiError.message || t('errors.unexpectedError'))
      })
    } finally {
      setLoading(false)
    }
  }, [t, selectedClubId, user?.memberships])

  // Load clubs on mount and when search or selectedClubId changes
  useEffect(() => {
    loadClubs(debouncedSearch)
  }, [loadClubs, debouncedSearch])

  const handleCreateClub = async () => {
    if (!createFormData.name.trim()) {
      toast.error(t('clubs.validation.nameRequired'))
      return
    }

    if (createFormData.name.length < 2) {
      toast.error(t('clubs.validation.nameMinLength'))
      return
    }

    if (createFormData.name.length > 80) {
      toast.error(t('clubs.validation.nameMaxLength'))
      return
    }

    try {
      setSubmitting(true)
      const clubData: CreateClubRequest = {
        name: createFormData.name.trim(),
        supportedSports: [DEFAULT_SPORT]
      }

      const club = await apiClient.createClub(clubData)

      setClubs(prev => [club, ...prev])
      setShowCreateDialog(false)
      setCreateFormData({ name: '' })
      toast.success(t('clubs.created'))

      // Refresh user memberships since the user was automatically added as admin
      await refreshUserMemberships()
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('errors.unexpectedError'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleEditClub = async () => {
    if (!editingClub) {
      return
    }

    if (!editFormData.name.trim()) {
      toast.error(t('clubs.validation.nameRequired'))
      return
    }

    if (editFormData.name.length < 2) {
      toast.error(t('clubs.validation.nameMinLength'))
      return
    }

    if (editFormData.name.length > 80) {
      toast.error(t('clubs.validation.nameMaxLength'))
      return
    }

    try {
      setSubmitting(true)
      const updateData: UpdateClubRequest = {
        name: editFormData.name.trim()
      }

      const updatedClub = await apiClient.updateClub(editingClub.id, updateData)

      setClubs(prev => prev.map(club =>
          club.id === editingClub.id ? updatedClub : club
      ))
      setShowEditDialog(false)
      setEditingClub(null)
      setEditFormData({ name: '' })
      toast.success(t('clubs.updated'))
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('errors.unexpectedError'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDeleteClub = async (club: Club) => {
    try {
      await apiClient.deleteClub(club.id)
      setClubs(prev => prev.filter(c => c.id !== club.id))
      toast.success(t('clubs.deleted'))
    } catch (error) {
      const apiError = error as ApiError
      toast.error(apiError.message || t('errors.unexpectedError'))
    }
  }

  const openCreateDialog = () => {
    setCreateFormData({ name: '' })
    setEditingClub(null)
    setShowCreateDialog(true)
  }

  const openEditDialog = (club: Club) => {
    setEditingClub(club)
    setEditFormData({
      name: club.name
    })
    setShowCreateDialog(false)
    setShowEditDialog(true)
  }

  const loadClubPlayers = useCallback(async (clubId: string) => {
    if (loadingClubPlayers[clubId] || clubPlayers[clubId]) {
      return // Already loading or loaded
    }

    setLoadingClubPlayers(prev => ({ ...prev, [clubId]: true }))

    try {
      const response = await apiClient.listPlayers({ clubId, pageSize: 100 })
      setClubPlayers(prev => ({
        ...prev,
        [clubId]: {
          count: response.items.length,
          players: response.items
        }
      }))
    } catch {
      setClubPlayers(prev => ({
        ...prev,
        [clubId]: {
          count: 0,
          players: []
        }
      }))
    } finally {
      setLoadingClubPlayers(prev => ({ ...prev, [clubId]: false }))
    }
  }, [loadingClubPlayers, clubPlayers])

  const openClubDetails = (club: Club) => {
    setSelectedClub(club)
    setShowClubDetails(true)
    loadClubPlayers(club.id)
  }

  const ClubCard = ({ club }: { club: Club }) => {
    const { isClubAdmin, isPlatformOwner } = useAuthStore()
    const playerData = clubPlayers[club.id]
    const isLoadingPlayers = loadingClubPlayers[club.id]
    const sports = club.supportedSports?.length ? club.supportedSports : [DEFAULT_SPORT]
    const seriesSports = club.seriesSports?.filter((sport) => sport !== 'SPORT_UNSPECIFIED') ?? []

    // Check if user can delete this club (club admin or platform owner)
    const canDeleteClub = isPlatformOwner() || isClubAdmin(club.id)

    // Load player count when card is rendered
    useEffect(() => {
      if (!playerData && !isLoadingPlayers) {
        loadClubPlayers(club.id)
      }
    }, [club.id, playerData, isLoadingPlayers])

    return (
        <Card className="hover:shadow-md transition-shadow" data-testid={`club-card-${club.id}`}>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div className="flex items-start space-x-3">
                <div className="p-2 bg-primary/10 rounded-lg">
                  <Buildings2 className="w-5 h-5 text-primary" />
                </div>
                <div className="flex-1">
                  <CardTitle className="text-lg" data-testid={`club-name-${club.id}`}>{club.name}</CardTitle>
                  <div className="flex items-center space-x-2 mt-1">
                    <div className="flex items-center space-x-1 text-sm text-muted-foreground">
                      <User size={14} />
                      <span>
                      {isLoadingPlayers ? '...' : `${playerData?.count || 0} ${t('players.title').toLowerCase()}`}
                    </span>
                    </div>
                    {playerData && playerData.count > 0 && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openClubDetails(club)}
                            className="h-6 px-2 text-xs text-primary hover:text-primary hover:bg-primary/10"
                            data-testid={`view-club-details-${club.id}`}
                        >
                          <Eye size={14} className="mr-1" />
                          {t('common.viewDetails')}
                        </Button>
                    )}
                  </div>
                  {seriesSports.length > 0 && (
                    <div className="flex items-center space-x-2 mt-2 text-xs text-muted-foreground">
                      <span>{t('clubs.seriesSports')}:</span>
                      <div className="flex items-center space-x-1">
                        {seriesSports.map((sport) => {
                          const Icon = sportIconComponent(sport)
                          return (
                            <span
                              key={sport}
                              className="inline-flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-primary"
                              title={t(sportTranslationKey(sport))}
                            >
                              <Icon className="h-4 w-4" aria-hidden />
                              <span className="sr-only">{t(sportTranslationKey(sport))}</span>
                            </span>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  <div className="flex flex-wrap items-center gap-2 mt-2 text-xs text-muted-foreground">
                    <span>{t('clubs.supportedSports')}:</span>
                    {sports.map((sport) => (
                      <Badge key={sport} variant="outline">
                        {t(sportTranslationKey(sport))}
                      </Badge>
                    ))}
                  </div>
                </div>
              </div>
              <ActionGroup>
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => openEditDialog(club)}
                    className="h-8 w-8 p-0"
                    data-testid={`edit-club-${club.id}`}
                >
                  <Edit2 size={16} />
                </Button>
                {canDeleteClub && (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                            data-testid={`delete-club-${club.id}`}
                        >
                          <Trash size={16} />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{t('clubs.deleteConfirm')}</AlertDialogTitle>
                          <AlertDialogDescription>
                            {t('clubs.deleteWarning')}
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                          <AlertDialogAction
                              onClick={() => handleDeleteClub(club)}
                              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                              data-testid={`confirm-delete-club-${club.id}`}
                          >
                            {t('common.delete')}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                )}
              </ActionGroup>
            </div>
          </CardHeader>
        </Card>
    )
  }

  const CreateClubForm = () => (
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="create-club-name">{t('clubs.name')} *</Label>
          <Input
              id="create-club-name"
              value={createFormData.name}
              onChange={(e) => setCreateFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder={t('clubs.name')}
              maxLength={80}
              autoFocus
              data-testid="create-club-name-input"
          />
        </div>
        <DialogFooter>
          <Button
              onClick={handleCreateClub}
              disabled={submitting || !createFormData.name.trim()}
              className="min-w-[80px]"
              data-testid="create-club-submit-button"
          >
            {submitting ? t('common.loading') : t('common.create')}
          </Button>
        </DialogFooter>
      </div>
  )

  const EditClubForm = () => (
      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="edit-club-name">{t('clubs.name')} *</Label>
          <Input
              id="edit-club-name"
              value={editFormData.name}
              onChange={(e) => setEditFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder={t('clubs.name')}
              maxLength={80}
              autoFocus
              data-testid="edit-club-name-input"
          />
        </div>
        <DialogFooter>
          <Button
              onClick={handleEditClub}
              disabled={submitting || !editFormData.name.trim()}
              className="min-w-[80px]"
              data-testid="edit-club-submit-button"
          >
            {submitting ? t('common.loading') : t('common.save')}
          </Button>
        </DialogFooter>
      </div>
  )

  return (
      <PageWrapper>
        {/* Header */}
        <PageHeaderSection>
          <HeaderContent>
            <h1 data-testid="clubs-title">{t('clubs.title')}</h1>
            <p>{t('clubs.subtitle')}</p>
          </HeaderContent>

          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button onClick={handleCreateClubClick} className="flex items-center space-x-2" data-testid="create-club-button">
                <Add size={18} color="white" />
                <span>{t('clubs.createNew')}</span>
              </Button>
            </DialogTrigger>
            <DialogContent data-testid="create-club-dialog">
              <DialogHeader>
                <DialogTitle>{t('clubs.createNew')}</DialogTitle>
                <DialogDescription>
                  Create a new table tennis club to organize players and tournaments.
                </DialogDescription>
              </DialogHeader>
              <CreateClubForm />
            </DialogContent>
          </Dialog>
        </PageHeaderSection>

        {/* Search */}
        <SearchSection>
          <SearchNormal1 size={18} color="var(--color-muted-foreground)" />
          <Input
              placeholder={t('clubs.searchClubs')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              data-testid="search-clubs-input"
          />
        </SearchSection>

        {/* Content */}
        {loading ? (
            <LoadingGrid>
              {[1, 2, 3, 4, 5, 6].map((i) => (
                  <Card key={i}>
                    <CardHeader>
                      <div className="flex items-start space-x-3">
                        <Skeleton className="w-10 h-10 rounded-lg" />
                        <div className="space-y-2 flex-1">
                          <Skeleton className="h-5 w-3/4" />
                          <Skeleton className="h-4 w-1/2" />
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <Skeleton className="h-4 w-1/3" />
                    </CardContent>
                  </Card>
              ))}
            </LoadingGrid>
        ) : clubs.length === 0 ? (
            <Card className="p-12">
              <SharedEmptyState>
                <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto">
                  <Buildings2 size={32} />
                </div>
                <h3>{t('clubs.noClubs')}</h3>
                <p>
                  {searchQuery ? `No clubs found matching "${searchQuery}"` : t('clubs.createFirst')}
                </p>
                {!searchQuery && (
                    <Button onClick={handleCreateClubClick} className="mt-4">
                      <Add size={18} color="white" className="mr-2" />
                      {t('clubs.createNew')}
                    </Button>
                )}
              </SharedEmptyState>
            </Card>
        ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '16px' }}>
              {clubs.map((club) => (
                  <Link to={`/clubs/${club.id}`} key={club.id}>
                    <ClubCard club={club} />
                  </Link>
              ))}
            </div>
        )}

        {/* Edit Dialog */}
        <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
          <DialogContent data-testid="edit-club-dialog">
            <DialogHeader>
              <DialogTitle>{t('clubs.edit')}</DialogTitle>
              <DialogDescription>
                Update the club information.
              </DialogDescription>
            </DialogHeader>
            <EditClubForm />
          </DialogContent>
        </Dialog>

        {/* Club Details Dialog */}
        <Dialog open={showClubDetails} onOpenChange={setShowClubDetails}>
          <DialogContent className="max-w-2xl" data-testid="club-details-dialog">
            <DialogHeader>
              <DialogTitle className="flex items-center space-x-2">
                <Buildings2 className="w-5 h-5" />
                <span>{selectedClub?.name}</span>
              </DialogTitle>
              <DialogDescription>
                {t('clubs.playersInClub')}
              </DialogDescription>
            </DialogHeader>
            <div className="max-h-96 overflow-y-auto">
              {selectedClub && clubPlayers[selectedClub.id] ? (
                  <div className="space-y-3">
                    <div className="text-sm text-muted-foreground mb-3">
                      {clubPlayers[selectedClub.id].count} {t('players.title').toLowerCase()}
                    </div>
                    {clubPlayers[selectedClub.id].players.length > 0 ? (
                        <div className="space-y-2">
                          {clubPlayers[selectedClub.id].players.map((player) => (
                              <div
                                  key={player.id}
                                  className="flex items-center justify-between p-3 border rounded-lg"
                              >
                                <div className="flex items-center space-x-3">
                                  <div className={`p-2 rounded-lg ${player.email ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>
                                    <User size={16} />
                                  </div>
                                  <div>
                                    <div className="font-medium">{player.displayName}</div>
                                    <div className="text-sm text-muted-foreground">
                                      {player.email ? (
                                          <span className="flex items-center space-x-1">
                                  <span>📧</span>
                                  <span>{player.email}</span>
                                  <span className="text-green-600 font-medium">(Authenticated)</span>
                                </span>
                                      ) : (
                                          <span className="flex items-center space-x-1">
                                  <span>👤</span>
                                  <span className="text-gray-600">(Club-created player)</span>
                                </span>
                                      )}
                                    </div>
                                  </div>
                                </div>
                                <div className="text-sm text-muted-foreground">
                                  {player.active ? (
                                      <span className="text-green-600">{t('common.active')}</span>
                                  ) : (
                                      <span className="text-gray-500">{t('common.inactive')}</span>
                                  )}
                                </div>
                              </div>
                          ))}
                        </div>
                    ) : (
                        <div className="text-center py-8 text-muted-foreground">
                          {t('players.noPlayers')}
                        </div>
                    )}
                  </div>
              ) : loadingClubPlayers[selectedClub?.id || ''] ? (
                  <div className="text-center py-8">
                    <div className="text-muted-foreground">{t('common.loading')}</div>
                  </div>
              ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    {t('players.noPlayers')}
                  </div>
              )}
            </div>
          </DialogContent>
        </Dialog>

        <ProfileCompletionModal
            isOpen={showProfileModal}
            onClose={() => setShowProfileModal(false)}
            onProfileUpdated={handleProfileUpdated}
            currentEmail={user?.email || ''}
        />
      </PageWrapper>
  )
}
