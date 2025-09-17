import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue
} from '@/components/ui/select'
import { apiClient } from '@/services/api'
import type { Player } from '@/types/api'
import { useState } from 'react'
import { toast } from 'sonner'

interface MergePlayersDialogProps {
  open: boolean
  onClose: () => void
  players: Player[]
  onMergeComplete: () => void
}

export function MergePlayersDialog({ open, onClose, players, onMergeComplete }: MergePlayersDialogProps) {
  const [targetPlayerId, setTargetPlayerId] = useState<string>('')
  const [sourcePlayerId, setSourcePlayerId] = useState<string>('')
  const [isLoading, setIsLoading] = useState(false)

  const handleMerge = async () => {
    if (!targetPlayerId || !sourcePlayerId) {
      toast.error('Please select both players to merge')
      return
    }

    if (targetPlayerId === sourcePlayerId) {
      toast.error('Cannot merge a player with themselves')
      return
    }

    setIsLoading(true)
    try {
      const result = await apiClient.mergePlayer(targetPlayerId, {
        sourcePlayerId: sourcePlayerId
      })

      toast.success(
        `Players merged successfully! Updated ${result.matchesUpdated} matches and ${result.tokensUpdated} tokens.`
      )
      
      onMergeComplete()
      onClose()
      
      // Reset form
      setTargetPlayerId('')
      setSourcePlayerId('')
    } catch (error: any) {
      toast.error(error?.message || 'Failed to merge players')
    } finally {
      setIsLoading(false)
    }
  }

  const handleClose = () => {
    if (!isLoading) {
      setTargetPlayerId('')
      setSourcePlayerId('')
      onClose()
    }
  }

  const targetPlayer = players.find(p => p.id === targetPlayerId)
  const sourcePlayer = players.find(p => p.id === sourcePlayerId)

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Merge Players</DialogTitle>
          <DialogDescription>
            Merge two player records into one. All matches and data from the source player will be transferred to the target player.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          <div className="space-y-2">
            <Label htmlFor="target-player">Target Player (Keep This One)</Label>
            <Select
              value={targetPlayerId}
              onValueChange={setTargetPlayerId}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select the player to keep..." />
              </SelectTrigger>
              <SelectContent>
                {players.map((player) => (
                  <SelectItem key={player.id} value={player.id}>
                    <div className="flex flex-col">
                      <span>{player.displayName}</span>
                      {player.email ? (
                        <span className="text-xs text-muted-foreground">✉️ {player.email}</span>
                      ) : (
                        <span className="text-xs text-muted-foreground">👤 No email (club-created)</span>
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {targetPlayer && (
              <div className="text-sm text-muted-foreground">
                This player will remain and receive all data from the source player.
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="source-player">Source Player (Delete After Merge)</Label>
            <Select
              value={sourcePlayerId}
              onValueChange={setSourcePlayerId}
              disabled={isLoading}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select the player to merge from..." />
              </SelectTrigger>
              <SelectContent>
                {players
                  .filter(p => p.id !== targetPlayerId)
                  .map((player) => (
                    <SelectItem key={player.id} value={player.id}>
                      <div className="flex flex-col">
                        <span>{player.displayName}</span>
                        {player.email ? (
                          <span className="text-xs text-muted-foreground">✉️ {player.email}</span>
                        ) : (
                          <span className="text-xs text-muted-foreground">👤 No email (club-created)</span>
                        )}
                      </div>
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
            {sourcePlayer && (
              <div className="text-sm text-muted-foreground">
                This player will be deleted and all their data transferred to the target player.
              </div>
            )}
          </div>

          {targetPlayer && sourcePlayer && (
            <div className="rounded-lg border p-4 space-y-2">
              <h4 className="font-medium">Merge Preview</h4>
              <div className="text-sm">
                <div className="flex items-center space-x-2">
                  <span className="text-green-600">✓ Keep:</span>
                  <span className="font-medium">{targetPlayer.displayName}</span>
                  {targetPlayer.email && <span className="text-muted-foreground">({targetPlayer.email})</span>}
                </div>
                <div className="flex items-center space-x-2">
                  <span className="text-red-600">✗ Delete:</span>
                  <span className="font-medium">{sourcePlayer.displayName}</span>
                  {sourcePlayer.email && <span className="text-muted-foreground">({sourcePlayer.email})</span>}
                </div>
              </div>
              <div className="text-xs text-muted-foreground">
                All matches, club memberships, and tokens will be transferred.
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button 
            onClick={handleMerge} 
            disabled={isLoading || !targetPlayerId || !sourcePlayerId}
            className="bg-red-600 hover:bg-red-700"
          >
            {isLoading ? 'Merging...' : 'Merge Players'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}