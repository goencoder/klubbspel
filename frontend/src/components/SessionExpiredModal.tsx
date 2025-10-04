import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/store/auth'

export function SessionExpiredModal() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { sessionExpired, dismissSessionExpired } = useAuthStore()

  const handleGoToLogin = () => {
    dismissSessionExpired()
    navigate('/login')
  }

  // Auto-dismiss if user navigates away
  useEffect(() => {
    if (!sessionExpired) {
      return
    }

    // If user closes modal without clicking button, still navigate after a short delay
    const timer = setTimeout(() => {
      if (sessionExpired) {
        dismissSessionExpired()
        navigate('/login')
      }
    }, 30000) // 30 seconds

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionExpired])

  return (
    <Dialog open={sessionExpired} onOpenChange={(open) => {
      if (!open) {
        handleGoToLogin()
      }
    }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 text-amber-500" />
            {t('auth.sessionExpired.title')}
          </DialogTitle>
          <DialogDescription>
            {t('auth.sessionExpired.message')}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button onClick={handleGoToLogin} className="w-full">
            {t('auth.sessionExpired.button')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
