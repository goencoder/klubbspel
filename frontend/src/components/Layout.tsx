import ClubNavigation from '@/components/ClubNavigation'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/store/auth'
import { Buildings2, Chart, Cup, LogoutCurve, People, Edit, User, ArrowDown2 } from 'iconsax-reactjs'
import { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate } from 'react-router-dom'

interface LayoutProps {
  children: ReactNode
}

export function Layout({ children }: LayoutProps) {
  const { t } = useTranslation()
  const location = useLocation()
  const navigate = useNavigate()
  const { user, isAuthenticated, logout } = useAuthStore()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const navigation = [
    {
      name: t('nav.series'),
      href: '/',
      icon: Cup,
      current: location.pathname === '/'
    },
    {
      name: t('nav.clubs'),
      href: '/clubs',
      icon: Buildings2,
      current: location.pathname === '/clubs'
    },
    {
      name: t('nav.players'),
      href: '/players',
      icon: People,
      current: location.pathname === '/players'
    },
    {
      name: t('nav.leaderboard'),
      href: '/leaderboard',
      icon: Chart,
      current: location.pathname === '/leaderboard'
    }
  ]

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border bg-card">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo */}
            <div className="flex items-center space-x-4">
              <Link to="/" className="flex items-center space-x-2">
                <Cup variant="Bold" className="w-8 h-8 text-primary" />
                <span className="text-xl font-bold text-foreground">Klubbspel</span>
              </Link>
            </div>

            {/* Navigation and Club Selector */}
            <div className="flex items-center space-x-4">
              <nav className="hidden md:flex space-x-1">
                {navigation.map((item) => {
                  const Icon = item.icon
                  return (
                    <Link
                      key={item.name}
                      to={item.href}
                      className={cn(
                        'flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                        item.current
                          ? 'bg-primary text-primary-foreground'
                          : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                      )}
                    >
                      <Icon size={18} className="text-current" />
                      <span>{item.name}</span>
                    </Link>
                  )
                })}
              </nav>

              {/* Club Navigation - only show if authenticated */}
              {isAuthenticated() && user && (
                <div className="hidden lg:block">
                  <ClubNavigation showAllOption={true} />
                </div>
              )}
            </div>

            {/* User Menu */}
            <div className="flex items-center space-x-4">
              {isAuthenticated() && user ? (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="flex items-center space-x-2">
                      <User size={18} />
                      <span className="hidden sm:inline">{user.displayName}</span>
                      <ArrowDown2 className="hidden sm:inline" size={16} />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem asChild>
                      <Link to="/settings" className="flex items-center space-x-2">
                        <Edit size={16} />
                        <span>{t('nav.settings')}</span>
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleLogout} className="flex items-center space-x-2">
                      <LogoutCurve size={16} />
                      <span>{t('auth.signOut')}</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : (
                <Button asChild>
                  <Link to="/login">{t('auth.signIn')}</Link>
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* Mobile Navigation */}
        <div className="md:hidden border-t border-border">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex space-x-1 py-2 overflow-x-auto">
              {navigation.map((item) => {
                const Icon = item.icon
                return (
                  <Link
                    key={item.name}
                    to={item.href}
                    className={cn(
                      'flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium whitespace-nowrap transition-colors',
                      item.current
                        ? 'bg-primary text-primary-foreground'
                        : 'text-muted-foreground hover:text-foreground hover:bg-muted'
                    )}
                  >
                    <Icon size={18} className="text-current" />
                    <span>{item.name}</span>
                  </Link>
                )
              })}

              {/* Club Navigation for mobile - show if authenticated and on clubs page */}
              {isAuthenticated() && user && location.pathname === '/clubs' && (
                <div className="flex items-center ml-2">
                  <ClubNavigation showAllOption={true} />
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
    </div>
  )
}