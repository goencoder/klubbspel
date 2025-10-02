import i18n from '@/i18n'
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AppState {
  language: string
  actor: string
  setLanguage: (language: string) => void
  setActor: (actor: string) => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      language: 'sv',
      actor: '',
      setLanguage: (language: string) => {
        set({ language })
  localStorage.setItem('klubbspel-language', language)
        // Use async/await to ensure language change is complete
        i18n.changeLanguage(language).then(() => {
          // Language changed successfully
        })
      },
      setActor: (actor: string) => set({ actor }),
    }),
    {
  name: 'klubbspel-settings',
    }
  )
)