import { useAppStore } from '@/store'
import { ReactNode, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

interface LanguageProviderProps {
  children: ReactNode
}

export function LanguageProvider({ children }: LanguageProviderProps) {
  const { language } = useAppStore()
  const { i18n } = useTranslation()
  const [isReady, setIsReady] = useState(i18n.isInitialized)

  useEffect(() => {
    const initLanguage = async () => {
      // Wait for i18n to be ready if not already initialized
      if (!i18n.isInitialized) {
        await new Promise<void>(resolve => {
          const checkInit = () => {
            if (i18n.isInitialized) {
              resolve()
            } else {
              setTimeout(checkInit, 10)
            }
          }
          checkInit()
        })
      }

      // Set the language from store
      if (language && i18n.language !== language) {
        console.log('LanguageProvider: changing language from', i18n.language, 'to', language)
        await i18n.changeLanguage(language)
      }

      setIsReady(true)
    }

    initLanguage()
  }, [language, i18n])

  if (!isReady) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-lg">Loading...</div>
      </div>
    )
  }

  return <>{children}</>
}