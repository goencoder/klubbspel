import { useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { SeriesRulesDialog } from '@/components/SeriesRulesDialog'
import type { SeriesFormat, LadderRules } from '@/types/api'
import { Info } from 'lucide-react'

export function RulesTestPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedFormat, setSelectedFormat] = useState<SeriesFormat>('SERIES_FORMAT_OPEN_PLAY')
  const [selectedLadderRules, setSelectedLadderRules] = useState<LadderRules | undefined>()

  const showRules = (format: SeriesFormat, ladderRules?: LadderRules) => {
    setSelectedFormat(format)
    setSelectedLadderRules(ladderRules)
    setDialogOpen(true)
  }

  return (
    <div className="container mx-auto p-8 max-w-4xl">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Series Rules Test</h1>
        <p className="text-gray-600">View rules for different series formats</p>
      </div>

      <div className="grid gap-4">
        {/* Free Play */}
        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xl font-semibold mb-2">Free Play</h3>
              <p className="text-sm text-gray-600">
                Play matches freely with any player. Rankings determined by ELO rating.
              </p>
            </div>
            <Button
              onClick={() => showRules('SERIES_FORMAT_OPEN_PLAY')}
              variant="outline"
              className="flex items-center gap-2"
            >
              <Info className="h-4 w-4" />
              View Rules
            </Button>
          </div>
        </Card>

        {/* Classic Ladder */}
        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xl font-semibold mb-2">Classic Ladder</h3>
              <p className="text-sm text-gray-600">
                Challenge any player. Winner improves position, loser keeps position (no penalty).
              </p>
            </div>
            <Button
              onClick={() => showRules('SERIES_FORMAT_LADDER', 'LADDER_RULES_CLASSIC')}
              variant="outline"
              className="flex items-center gap-2"
            >
              <Info className="h-4 w-4" />
              View Rules
            </Button>
          </div>
        </Card>

        {/* Aggressive Ladder */}
        <Card className="p-6">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-xl font-semibold mb-2">Aggressive Ladder</h3>
              <p className="text-sm text-gray-600">
                Challenge any player. Winner improves position, loser drops one position (penalty).
              </p>
            </div>
            <Button
              onClick={() => showRules('SERIES_FORMAT_LADDER', 'LADDER_RULES_AGGRESSIVE')}
              variant="outline"
              className="flex items-center gap-2"
            >
              <Info className="h-4 w-4" />
              View Rules
            </Button>
          </div>
        </Card>
      </div>

      <SeriesRulesDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        format={selectedFormat}
        ladderRules={selectedLadderRules}
      />
    </div>
  )
}
