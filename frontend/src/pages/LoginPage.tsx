import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuthStore } from '@/store/auth'
import { CheckCircle, LogIn, Mail } from 'lucide-react'
import React, { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'

export function LoginPage() {
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

  // Check for token in URL on mount
  useEffect(() => {
    const token = searchParams.get('token') || searchParams.get('apikey')
    if (token) {
      handleTokenValidation(token)
    }
  }, [searchParams])

  // Redirect if already authenticated
  useEffect(() => {
    if (isAuthenticated() && user) {
      const returnTo = searchParams.get('returnTo') || searchParams.get('return_url') || '/'
      navigate(returnTo, { replace: true })
    }
  }, [isAuthenticated, user, navigate, searchParams])

  const handleTokenValidation = async (token: string) => {
    try {
      setIsLoading(true)
      await validateToken(token)
      toast.success('Welcome! You have been successfully logged in.')
    } catch (error) {
      console.error('Token validation failed:', error)
      toast.error('Invalid or expired login link. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  const handleSendMagicLink = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!email.trim()) {
      toast.error('Please enter your email address')
      return
    }

    try {
      setIsLoading(true)
      clearError()

      const returnUrl = window.location.origin + '/login' +
        (searchParams.get('returnTo') ? `?returnTo=${encodeURIComponent(searchParams.get('returnTo')!)}` : '')

      await sendMagicLink(email.trim(), returnUrl)
      setMagicLinkSent(true)
      toast.success('Magic link sent! Check your email.')
    } catch (error) {
      console.error('Failed to send magic link:', error)
      toast.error('Failed to send magic link. Please try again.')
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
            Welcome to Klubbspel
          </CardTitle>
          <CardDescription>
            Sign in to manage your table tennis tournaments and track your progress
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
                <h3 className="font-semibold">Check your email</h3>
                <p className="text-sm text-muted-foreground">
                  We've sent a magic link to <strong>{email}</strong>.
                  Click the link in your email to sign in.
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
                <Label htmlFor="email">Email Address</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <Input
                    id="email"
                    type="email"
                    placeholder="your@email.com"
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
                {isLoading ? 'Sending...' : 'Send Magic Link'}
              </Button>

              <div className="text-center text-sm text-muted-foreground">
                <p>
                  We'll send you a secure link to sign in without a password.
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
