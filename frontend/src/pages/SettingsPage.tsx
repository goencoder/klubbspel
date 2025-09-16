import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import apiClient from '@/services/api'
import { useAppStore } from '@/store'
import { useAuthStore } from '@/store/auth'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

export function SettingsPage() {
  const { t, i18n } = useTranslation()
  const { language, setLanguage } = useAppStore()
  const { user, refreshUser } = useAuthStore()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false)

  useEffect(() => {
    if (user) {
      // Get the first and last name directly from the backend response via auth store
      // We need to call the validate token again to get the fresh user data with firstName/lastName
      // For now, let's extract from response or leave empty if displayName is just email
      const isEmailDisplayName = user.displayName.includes('@')
      if (isEmailDisplayName) {
        // If displayName is email, that means firstName/lastName are empty
        setFirstName('')
        setLastName('')
      } else {
        // Parse displayName that was constructed from firstName + lastName
        const names = user.displayName.split(' ')
        setFirstName(names[0] || '')
        setLastName(names.slice(1).join(' ') || '')
      }
    }
  }, [user])

  const handleLanguageChange = async (newLanguage: string) => {
    try {
      setLanguage(newLanguage)
      await i18n.changeLanguage(newLanguage)
      toast.success(t('common.success'))
    } catch (error) {
      toast.error('Failed to change language')
    }
  }

  const handleProfileSave = async () => {
    if (!user || !firstName.trim() || !lastName.trim()) {
      toast.error(t('settings.profile.validation.required'))
      return
    }

    setIsUpdatingProfile(true)
    try {
      await apiClient.updateProfile({
        firstName: firstName.trim(),
        lastName: lastName.trim()
      })
      
      // Refresh user data to get updated display name
      await refreshUser()
      
      toast.success(t('settings.profile.updated'))
    } catch (error) {
      toast.error(t('settings.profile.update.failed'))
    } finally {
      setIsUpdatingProfile(false)
    }
  }

  const isProfileChanged = user && (
    firstName.trim() !== (user.displayName.split(' ')[0] || '') ||
    lastName.trim() !== (user.displayName.split(' ').slice(1).join(' ') || '')
  )

  const isProfileComplete = firstName.trim() && lastName.trim()
  const isProfileRequired = user && (!user.displayName || user.displayName === user.email)

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-foreground">{t('settings.title')}</h1>
        <p className="text-muted-foreground mt-2">
          {t('settings.actor.description')}
        </p>
      </div>

      <div className="space-y-6">
        {/* Language Settings */}
        <Card>
          <CardHeader>
            <CardTitle>{t('settings.language')}</CardTitle>
            <CardDescription>
              {t('settings.language.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <Label htmlFor="language-select">{t('settings.language')}</Label>
              <Select value={language || undefined} onValueChange={handleLanguageChange}>
                <SelectTrigger id="language-select" className="w-full max-w-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sv">{t('settings.language.sv')}</SelectItem>
                  <SelectItem value="en">{t('settings.language.en')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Profile Settings */}
        <Card className={isProfileRequired ? "border-red-200" : ""}>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {t('settings.profile')}
              {isProfileRequired && (
                <span className="text-red-500 text-sm font-normal">
                  ({t('settings.profile.required')})
                </span>
              )}
            </CardTitle>
            <CardDescription>
              {t('settings.profile.description')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="first-name-input" className="flex items-center gap-1">
                    {t('settings.profile.firstName')}
                    <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="first-name-input"
                    type="text"
                    placeholder={t('settings.profile.firstName.placeholder')}
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    className={`max-w-md ${!firstName.trim() && isProfileRequired ? 'border-red-300 focus:border-red-500' : ''}`}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="last-name-input" className="flex items-center gap-1">
                    {t('settings.profile.lastName')}
                    <span className="text-red-500">*</span>
                  </Label>
                  <Input
                    id="last-name-input"
                    type="text"
                    placeholder={t('settings.profile.lastName.placeholder')}
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    className={`max-w-md ${!lastName.trim() && isProfileRequired ? 'border-red-300 focus:border-red-500' : ''}`}
                  />
                </div>
              </div>
              {isProfileRequired && !isProfileComplete && (
                <div className="text-red-600 text-sm flex items-center gap-2">
                  <span className="font-medium">⚠️</span>
                  {t('settings.profile.completion.required')}
                </div>
              )}
              <Button
                onClick={handleProfileSave}
                disabled={!isProfileChanged || isUpdatingProfile || !isProfileComplete}
                className="w-fit"
              >
                {isUpdatingProfile ? t('common.saving') : t('common.save')}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}