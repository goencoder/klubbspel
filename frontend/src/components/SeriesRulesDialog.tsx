import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { apiClient } from '@/services/api'
import type { RulesDescription, SeriesFormat, LadderRules } from '@/types/api'
import { AlertCircle, CheckCircle2, Info } from 'lucide-react'

interface SeriesRulesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  format: SeriesFormat
  ladderRules?: LadderRules
}

export function SeriesRulesDialog({
  open,
  onOpenChange,
  format,
  ladderRules
}: SeriesRulesDialogProps) {
  const { t } = useTranslation()
  const [rules, setRules] = useState<RulesDescription | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchRules = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await apiClient.getSeriesRules({
        format,
        ladderRules
      })
      setRules(response.rules)
    } catch (err) {
      setError(t('series.rules.error'))
      // Error logged for debugging - consider replacing with proper error tracking service in production
      // eslint-disable-next-line no-console
      console.error('Error fetching rules:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open && format !== 'SERIES_FORMAT_UNSPECIFIED') {
      fetchRules()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, format, ladderRules])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Info className="h-5 w-5" />
            {loading ? t('series.rules.loading') : rules?.title || t('series.rules.title')}
          </DialogTitle>
          <DialogDescription>
            {rules?.summary || ''}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh] pr-4">
          {loading && (
            <div className="flex items-center justify-center py-8">
              <LoadingSpinner />
            </div>
          )}

          {error && (
            <div className="flex items-center gap-2 p-4 bg-red-50 text-red-700 rounded-lg">
              <AlertCircle className="h-5 w-5" />
              <span>{error}</span>
            </div>
          )}

          {rules && !loading && (
            <div className="space-y-6">
              {/* Rules List */}
              <div>
                <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                  <CheckCircle2 className="h-5 w-5 text-green-600" />
                  {t('series.rules.title')}
                </h3>
                <ul className="space-y-2">
                  {rules.rules.map((rule, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <span className="text-blue-600 font-bold min-w-[24px]">
                        {index + 1}.
                      </span>
                      <span className="text-gray-700">{rule}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Examples */}
              {rules.examples && rules.examples.length > 0 && (
                <div>
                  <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                    <Info className="h-5 w-5 text-blue-600" />
                    {t('series.rules.examples')}
                  </h3>
                  <div className="space-y-4">
                    {rules.examples.map((example, index) => (
                      <div
                        key={index}
                        className="border-l-4 border-blue-500 bg-blue-50 p-4 rounded-r-lg"
                      >
                        <p className="font-medium text-gray-900 mb-1">
                          {example.scenario}
                        </p>
                        <p className="text-sm text-gray-600">
                          <span className="font-semibold">Result:</span> {example.outcome}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </ScrollArea>

        <div className="flex justify-end pt-4 border-t">
          <Button onClick={() => onOpenChange(false)} variant="outline">
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
