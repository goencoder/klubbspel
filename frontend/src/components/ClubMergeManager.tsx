import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { apiClient } from '@/services/api'
import type { MergeCandidate } from '@/types/api'
import { useAuthStore } from '@/store/auth'
import { useState, useEffect, useCallback } from 'react'
import { toast } from 'sonner'
import { User, Mail, UserPlus } from 'lucide-react'

interface ClubMergeManagerProps {
  clubId: string
  onMergeComplete?: () => void
}

export function ClubMergeManager({ clubId, onMergeComplete }: ClubMergeManagerProps) {
  const { user } = useAuthStore()
  const [mergeCandidates, setMergeCandidates] = useState<MergeCandidate[]>([])
  const [loading, setLoading] = useState(false)
  const [showMergeDialog, setShowMergeDialog] = useState(false)
  const [selectedCandidate, setSelectedCandidate] = useState<MergeCandidate | null>(null)
  const [merging, setMerging] = useState(false)

  const canUseMergeFeature = user && user.playerId

  const loadMergeCandidates = useCallback(async () => {
    if (!canUseMergeFeature) {
      return
    }
    
    setLoading(true)
    try {
      const response = await apiClient.findMergeCandidates({ clubId })
      setMergeCandidates(response.candidates || [])
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to load merge candidates:', error)
      toast.error('Failed to load merge candidates')
    } finally {
      setLoading(false)
    }
  }, [clubId, canUseMergeFeature])

  useEffect(() => {
    loadMergeCandidates()
  }, [loadMergeCandidates])

  const handleMergeCandidate = async (candidate: MergeCandidate) => {
    if (!user?.playerId) {
      toast.error('You must be logged in to merge players')
      return
    }

    setMerging(true)
    try {
      await apiClient.mergePlayer(user.playerId, {
        sourcePlayerId: candidate.player.id,
      })
      
      toast.success(`Successfully merged ${candidate.player.displayName} into your account`)
      setShowMergeDialog(false)
      setSelectedCandidate(null)
      
      // Reload candidates to reflect changes
      await loadMergeCandidates()
      
      // Notify parent component if provided
      onMergeComplete?.()
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Merge failed:', error)
      toast.error('Failed to merge player')
    } finally {
      setMerging(false)
    }
  }

  const confirmMerge = (candidate: MergeCandidate) => {
    setSelectedCandidate(candidate)
    setShowMergeDialog(true)
  }

  if (!canUseMergeFeature) {
    return null
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <UserPlus className="h-5 w-5" />
            <span>{t('clubs.detail.merge.title')}</span>
          </CardTitle>
          <CardDescription>
            {t('clubs.detail.merge.description')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="text-muted-foreground">{t('clubs.detail.merge.loadingMatches')}</div>
            </div>
          ) : mergeCandidates.length === 0 ? (
            <div className="text-center py-8">
              <User className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">
                {t('clubs.detail.merge.noMatches')}
              </p>
              <p className="text-sm text-muted-foreground mt-2">
                {t('clubs.detail.merge.noMatchesHelp')}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground mb-4">
                We found {mergeCandidates.length} potential match{mergeCandidates.length !== 1 ? 'es' : ''} for your account:
              </p>
              
              {mergeCandidates.map((candidate) => (
                <div
                  key={candidate.player.id}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50"
                >
                  <div className="flex items-center space-x-3">
                    <div className="h-10 w-10 bg-muted rounded-full flex items-center justify-center">
                      <User className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <div className="font-medium">{candidate.player.displayName}</div>
                      <div className="flex items-center space-x-2 text-sm text-muted-foreground">
                        <Badge variant="secondary" className="text-xs">
                          No email (club-created)
                        </Badge>
                        <Badge variant="outline" className="text-xs bg-blue-50 text-blue-700">
                          {Math.round(candidate.similarityScore * 100)}% match
                        </Badge>
                        {candidate.player.clubMemberships?.length && (
                          <span>• Member since {new Date(candidate.player.clubMemberships[0].joinedAt).toLocaleDateString()}</span>
                        )}
                      </div>
                    </div>
                  </div>
                  
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => confirmMerge(candidate)}
                    className="ml-4"
                  >
                    Merge Into My Account
                  </Button>
                </div>
              ))}

              <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-950/20 rounded-lg">
                <p className="text-sm text-blue-800 dark:text-blue-200">
                  <strong>What happens when you merge:</strong>
                </p>
                <ul className="text-sm text-blue-700 dark:text-blue-300 mt-1 space-y-1">
                  <li>• All match history will be transferred to your account</li>
                  <li>• Club memberships will be combined</li>
                  <li>• The old account will be deleted</li>
                  <li>• This action cannot be undone</li>
                </ul>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Merge Confirmation Dialog */}
      <Dialog open={showMergeDialog} onOpenChange={setShowMergeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Account Merge</DialogTitle>
            <DialogDescription>
              Are you sure you want to merge this account into yours?
            </DialogDescription>
          </DialogHeader>
          
          {selectedCandidate && (
            <div className="space-y-4">
              <div className="p-4 border rounded-lg">
                <h4 className="font-medium mb-2">Merging:</h4>
                <div className="flex items-center space-x-3">
                  <User className="h-8 w-8 text-muted-foreground" />
                  <div>
                    <div className="font-medium">{selectedCandidate.player.displayName}</div>
                    <Badge variant="secondary" className="text-xs">No email (club-created)</Badge>
                  </div>
                </div>
              </div>
              
              <div className="p-4 border rounded-lg">
                <h4 className="font-medium mb-2">Into your account:</h4>
                <div className="flex items-center space-x-3">
                  <Mail className="h-8 w-8 text-muted-foreground" />
                  <div>
                    <div className="font-medium">{user?.displayName}</div>
                    <div className="text-sm text-muted-foreground">{user?.email}</div>
                  </div>
                </div>
              </div>

              <div className="bg-amber-50 dark:bg-amber-950/20 p-3 rounded-lg">
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  <strong>Warning:</strong> This action cannot be undone. All data from the club-created account will be permanently transferred to your authenticated account.
                </p>
              </div>
            </div>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowMergeDialog(false)}
              disabled={merging}
            >
              Cancel
            </Button>
            <Button
              onClick={() => selectedCandidate && handleMergeCandidate(selectedCandidate)}
              disabled={merging}
              className="bg-red-600 hover:bg-red-700"
            >
              {merging ? 'Merging...' : 'Confirm Merge'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}