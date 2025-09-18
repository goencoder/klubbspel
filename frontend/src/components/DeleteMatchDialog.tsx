import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { apiClient } from '@/services/api'
import type { MatchView } from '@/types/api'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

interface DeleteMatchDialogProps {
  match: MatchView | null
  isOpen: boolean
  onClose: () => void
  onMatchDeleted: (matchId: string) => void
}

export function DeleteMatchDialog({
  match,
  isOpen,
  onClose,
  onMatchDeleted,
}: DeleteMatchDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)

  const handleDelete = async () => {
    if (!match) return

    try {
      setLoading(true)
      await apiClient.deleteMatch({ matchId: match.id })
      onMatchDeleted(match.id)
      toast.success(t('matches.deleted'))
      onClose()
    } catch (error: unknown) {
      toast.error((error as Error).message || t('error.generic'))
    } finally {
      setLoading(false)
    }
  }

  if (!match) return null

  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('matches.delete')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('matches.deleteConfirm')}
            <br />
            <strong>
              {match.playerAName} {match.scoreA} - {match.scoreB} {match.playerBName}
            </strong>
            <br />
            {new Date(match.playedAt).toLocaleDateString()}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
          <AlertDialogAction onClick={handleDelete} disabled={loading}>
            {loading ? t('common.loading') : t('matches.delete')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}