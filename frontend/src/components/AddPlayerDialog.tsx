import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { LoadingSpinner } from '@/components/LoadingSpinner'
import { toast } from 'sonner'
import { apiClient } from '@/services/api'

interface AddPlayerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clubId: string
  clubName: string
  onPlayerAdded?: () => void
}

interface AddPlayerFormData {
  firstName: string
  lastName: string
  email: string
}

export function AddPlayerDialog({
  open,
  onOpenChange,
  clubId: _clubId,
  clubName,
  onPlayerAdded,
}: AddPlayerDialogProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [_error, _setError] = useState('')
  const [formData, setFormData] = useState<AddPlayerFormData>({
    firstName: '',
    lastName: '',
    email: '',
  })
  const [errors, setErrors] = useState<Partial<AddPlayerFormData>>({})

  const validateForm = (): boolean => {
    const newErrors: Partial<AddPlayerFormData> = {}
    
    if (!formData.firstName.trim()) {
      newErrors.firstName = t('clubs.members.firstNameRequired')
    }
    
    if (!formData.lastName.trim()) {
      newErrors.lastName = t('clubs.members.lastNameRequired')
    }
    
    // Email validation (only if email is provided)
    if (formData.email.trim()) {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      if (!emailRegex.test(formData.email.trim())) {
        newErrors.email = t('clubs.members.invalidEmail')
      }
    }
    
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!validateForm()) {
      return
    }

    setLoading(true)
    try {
      const response = await apiClient.addPlayerToClub({
        clubId: _clubId,
        firstName: formData.firstName.trim(),
        lastName: formData.lastName.trim(),
        email: formData.email.trim() || undefined,
      })

      if (response.success) {
        toast.success(t('clubs.members.playerAdded'))
        setFormData({ firstName: '', lastName: '', email: '' })
        setErrors({})
        onPlayerAdded?.()
        onOpenChange(false)
      } else {
        toast.error(t('clubs.members.addFailed'))
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : t('clubs.members.addFailed')
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }

  const handleInputChange = (field: keyof AddPlayerFormData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: undefined }))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('clubs.members.addPlayerToClub', { clubName })}</DialogTitle>
          <DialogDescription>
            Add a new player to this club. If an email is provided, they will be notified.
          </DialogDescription>
        </DialogHeader>
        
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="firstName">{t('clubs.members.firstName')} *</Label>
            <Input
              id="firstName"
              value={formData.firstName}
              onChange={(e) => handleInputChange('firstName', e.target.value)}
              placeholder="Enter first name"
              disabled={loading}
              required
            />
            {errors.firstName && (
              <p className="text-sm text-red-600">{errors.firstName}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="lastName">{t('clubs.members.lastName')} *</Label>
            <Input
              id="lastName"
              value={formData.lastName}
              onChange={(e) => handleInputChange('lastName', e.target.value)}
              placeholder="Enter last name"
              disabled={loading}
              required
            />
            {errors.lastName && (
              <p className="text-sm text-red-600">{errors.lastName}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="email">{t('clubs.members.emailOptional')}</Label>
            <Input
              id="email"
              type="email"
              value={formData.email}
              onChange={(e) => handleInputChange('email', e.target.value)}
              placeholder="Enter email address (optional)"
              disabled={loading}
            />
            {errors.email && (
              <p className="text-sm text-red-600">{errors.email}</p>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading && <LoadingSpinner className="mr-2 h-4 w-4" />}
              {loading ? t('clubs.members.addingPlayer') : t('clubs.members.addPlayer')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}