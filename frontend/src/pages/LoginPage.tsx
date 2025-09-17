import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuthStore } from '@/store/auth'
import { CheckCircle, LogIn, Mail } from 'lucide-react'
import React, {useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'

export function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState('')
  const [magicLinkSent, setMagicLinkSent] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  const {
    sendMagicLink,
    validateToken,
    isAuthenticated,
    user,
    error,
    clearError
  } = useAuthStore()

  const handleTokenValidation = useCallback(async (token: string) => {
    try {
      setIsLoading(true)
      await validateToken(token)
      toast.success(t('login.welcome'))
    } catch (_error) {
      toast.error(t('login.invalidLink'))
    } finally {
      setIsLoading(false)
    }
  }, [validateToken, t])

  // Check for token in URL on mount
  useEffect(() => {
    const token = searchParams.get('token') || searchParams.get('apikey')
    if (token) {
      handleTokenValidation(token)
    }
  }, [searchParams, handleTokenValidation])

  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated() && user) {
      const returnTo = searchParams.get('returnTo') || searchParams.get('return_url') || '/'
      navigate(returnTo, { replace: true })
    }
  }, [isAuthenticated, user, navigate, searchParams])

  const handleSendMagicLink = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!email.trim()) {
      toast.error('Please enter your email address')
      return
    }

    try {
      setIsLoading(true)
      clearError()

      const returnTo = searchParams.get('returnTo')
      const returnUrl = returnTo 
        ? `${window.location.origin}/login?returnTo=${encodeURIComponent(returnTo)}`
        : '/leaderboard'

      await sendMagicLink(email.trim(), returnUrl)
      setMagicLinkSent(true)
      toast.success(t('login.magicLinkSent'))
    } catch (_error) {
      toast.error(t('login.magicLinkFailed'))
    } finally {
      setIsLoading(false)
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardContent className="pt-6">
            <div className="text-center space-y-4">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
              <p>Logging you in...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="flex items-center justify-center gap-2 text-2xl">
            <LogIn className="h-6 w-6" />
            {t('login.title')}
          </CardTitle>
          <CardDescription>
            {t('login.description')}
          </CardDescription>
        </CardHeader>

        <CardContent>
          {magicLinkSent ? (
            <div className="text-center space-y-4">
              <div className="flex justify-center">
                <div className="rounded-full bg-green-100 p-3">
                  <CheckCircle className="h-8 w-8 text-green-600" />
                </div>
              </div>

              <div className="space-y-2">
                <h3 className="font-semibold">{t('login.checkEmail')}</h3>
                <p className="text-sm text-muted-foreground">
                  {t('login.checkEmailDescription')} <strong>{email}</strong>.
                  {t('login.checkEmailInstructions')}
                </p>
              </div>

              <div className="pt-4">
                <Button
                  variant="outline"
                  onClick={() => {
                    setMagicLinkSent(false)
                    setEmail('')
                  }}
                  className="w-full"
                >
                  Use different email
                </Button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSendMagicLink} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t('login.emailAddress')}</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="email"
                    type="email"
                    placeholder={t('login.emailPlaceholder')}
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="pl-10"
                    required
                    disabled={isLoading}
                  />
                </div>
              </div>

              {error && (
                <div className="text-sm text-red-600 bg-red-50 p-3 rounded-md">
                  {error}
                </div>
              )}

              <Button
                type="submit"
                className="w-full"
                disabled={isLoading}
              >
                {isLoading ? t('login.sending') : t('login.sendMagicLink')}
              </Button>

              <div className="text-center text-sm text-muted-foreground">
                <p>
                  {t('login.description2')}
                </p>
              </div>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default LoginPage
