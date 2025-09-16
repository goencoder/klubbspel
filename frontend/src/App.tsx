import { LanguageProvider } from '@/components/LanguageProvider'
import { Layout } from '@/components/Layout'
import { Toaster } from '@/components/ui/sonner'
import { ClubDetailPage } from '@/pages/ClubDetailPage'
import { ClubsPage } from '@/pages/ClubsPage'
import { LeaderboardPage } from '@/pages/LeaderboardPage'
import { LoginPage } from '@/pages/LoginPage'
import { PlayersPage } from '@/pages/PlayersPage'
import { SeriesDetailPage } from '@/pages/SeriesDetailPage'
import { SeriesListPage } from '@/pages/SeriesListPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { useAppStore } from '@/store'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Route, BrowserRouter as Router, Routes } from 'react-router-dom'
import './i18n'

function App() {
  const { language } = useAppStore()
  const { i18n } = useTranslation()

  // Sync language changes
  useEffect(() => {
    if (language && i18n.language !== language) {
      console.log('App: changing language from', i18n.language, 'to', language)
      i18n.changeLanguage(language)
    }
  }, [language, i18n])

  return (
    <LanguageProvider>
      <Router>
        <Layout>
          <Routes>
            <Route path="/" element={<SeriesListPage />} />
            <Route path="/clubs" element={<ClubsPage />} />
            <Route path="/clubs/:id" element={<ClubDetailPage />} />
            <Route path="/series/:id" element={<SeriesDetailPage />} />
            <Route path="/series/:id/leaderboard" element={<LeaderboardPage />} />
            <Route path="/players" element={<PlayersPage />} />
            <Route path="/leaderboard" element={<LeaderboardPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/auth/login" element={<LoginPage />} />
          </Routes>
        </Layout>
        <Toaster position="top-right" />
      </Router>
    </LanguageProvider>
  )
}

export default App