import { LoadingSpinner } from '@/components/LoadingSpinner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import type { Player } from '@/types/api'
import { People, TickCircle } from 'iconsax-reactjs'
import { useTranslation } from 'react-i18next'

interface PlayerConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  playerName: string
  similarPlayers: Player[]
  onUseSimilar: (player: Player) => void
  onCreateNew: () => void
  loading?: boolean
}

export function PlayerConfirmDialog({
  open,
  onOpenChange,
  playerName,
  similarPlayers,
  onUseSimilar,
  onCreateNew,
  loading = false
}: PlayerConfirmDialogProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center space-x-2">
            <People size={20} />
            <span>{t('players.similarFound')}</span>
          </DialogTitle>
          <DialogDescription>
            {t('players.similarMessage')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="p-4 bg-muted rounded-lg">
            <p className="text-sm font-medium">
              Creating player: <span className="font-bold">{playerName}</span>
            </p>
          </div>

          <div className="space-y-3">
            <h4 className="font-medium text-sm text-muted-foreground">Similar players found:</h4>
            <div className="grid gap-3 max-h-[300px] overflow-y-auto">
              {similarPlayers.map((player) => (
                <Card key={player.id} className="cursor-pointer hover:bg-muted/50 transition-colors">
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base">{player.display_name}</CardTitle>
                      <Badge variant={player.active ? 'default' : 'secondary'}>
                        {player.active ? t('common.active') : t('common.inactive')}
                      </Badge>
                    </div>
                    <CardDescription>
                      <div className="text-sm text-foreground">
                        {t('players.club')}: {player.clubId}
                      </div>
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <div className="flex justify-between items-center">
                      <div className="text-xs text-muted-foreground">
                        {player.created_at ? `${t('common.joined')} ${new Date(player.created_at).toLocaleDateString()}` : t('common.unknown') + ' join date'}
                      </div>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => onUseSimilar(player)}
                        className="ml-auto"
                      >
                        <TickCircle size={14} className="mr-1" />
                        {t('players.useExisting')}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        </div>

        <DialogFooter className="flex flex-col sm:flex-row gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} className="order-3 sm:order-1">
            {t('common.cancel')}
          </Button>
          <Button
            variant="default"
            onClick={onCreateNew}
            disabled={loading}
            className="order-2"
          >
            {loading ? <LoadingSpinner size="sm" /> : t('players.createNew')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}