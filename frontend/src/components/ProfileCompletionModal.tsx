import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import apiClient from '@/services/api'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

interface ProfileCompletionModalProps {
  isOpen: boolean
  onClose: () => void
  onProfileUpdated: () => void
  currentEmail: string
}

export function ProfileCompletionModal({ 
  isOpen, 
  onClose, 
  onProfileUpdated, 
  currentEmail 
}: ProfileCompletionModalProps) {
  const { t } = useTranslation()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [isUpdating, setIsUpdating] = useState(false)

  const handleSave = async () => {
    if (!firstName.trim() || !lastName.trim()) {
      toast.error(t('settings.profile.validation.required'))
      return
    }

    setIsUpdating(true)
    try {
      await apiClient.updateProfile({
        firstName: firstName.trim(),
        lastName: lastName.trim()
      })
      toast.success(t('settings.profile.updated'))
      onProfileUpdated()
      onClose()
    } catch (_error) {
      toast.error(t('settings.profile.update.failed'))
    } finally {
      setIsUpdating(false)
    }
  }

  const handleCancel = () => {
    setFirstName('')
    setLastName('')
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={handleCancel}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('profile.completion.modal.title')}</DialogTitle>
          <DialogDescription>
            {t('profile.completion.modal.description')}
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4 py-4">
          <div className="text-sm text-muted-foreground">
            <strong>{t('profile.completion.modal.email')}:</strong> {currentEmail}
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="modal-first-name" className="flex items-center gap-1">
              {t('settings.profile.firstName')}
              <span className="text-red-500">*</span>
            </Label>
            <Input
              id="modal-first-name"
              type="text"
              placeholder={t('settings.profile.firstName.placeholder')}
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              className={!firstName.trim() ? 'border-red-300 focus:border-red-500' : ''}
            />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="modal-last-name" className="flex items-center gap-1">
              {t('settings.profile.lastName')}
              <span className="text-red-500">*</span>
            </Label>
            <Input
              id="modal-last-name"
              type="text"
              placeholder={t('settings.profile.lastName.placeholder')}
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              className={!lastName.trim() ? 'border-red-300 focus:border-red-500' : ''}
            />
          </div>
          
          <div className="text-red-600 text-sm flex items-center gap-2">
            <span className="font-medium">⚠️</span>
            {t('settings.profile.completion.required')}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel} disabled={isUpdating}>
            {t('common.cancel')}
          </Button>
          <Button 
            onClick={handleSave} 
            disabled={!firstName.trim() || !lastName.trim() || isUpdating}
          >
            {isUpdating ? t('common.saving') : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}