import type { BackendModule } from 'i18next'

const loaders: Record<
  string,
  () => Promise<{ default: { translation: Record<string, string> } }>
> = {
  zhCN: () => import('./locales/zh.json'),
  zhTW: () => import('./locales/zh-TW.json'),
  fr: () => import('./locales/fr.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
}

export const localeBackend: BackendModule = {
  type: 'backend',
  init() {},
  async read(
    language: string,
    namespace: string
  ): Promise<Record<string, string>> {
    const load = loaders[language]
    if (!load || namespace !== 'translation') {
      return {}
    }
    const resource = await load()
    return resource.default.translation
  },
}
