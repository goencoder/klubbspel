import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import enTranslations from './locales/en.json'
import svTranslations from './locales/sv.json'

const resources = {
  sv: { translation: svTranslations },
  en: { translation: enTranslations }
}

// Get stored language safely
let storedLanguage = 'sv'
if (typeof window !== 'undefined') {
  storedLanguage = localStorage.getItem('klubbspel-language') || 'sv'
  console.log('i18n: Loading stored language:', storedLanguage)
}

// Initialize i18n synchronously
i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: storedLanguage,
    fallbackLng: 'sv',
    debug: false,
    interpolation: {
      escapeValue: false
    },
    react: {
      useSuspense: false // Disable suspense for better initialization
    }
  })

export default i18n