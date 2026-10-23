import { act, render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { I18nextProvider, useTranslation } from 'react-i18next'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

beforeEach(() => {
  vi.resetModules()
  localStorage.clear()
  for (const language of ['en', 'zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi']) {
    i18next.removeResourceBundle(language, 'translation')
  }
})

afterEach(() => {
  localStorage.clear()
  vi.doUnmock('../locales/ru.json')
})

test.each([
  ['zh-CN', 'zhCN'],
  ['zh-TW', 'zhTW'],
  ['fr', 'fr'],
  ['ru', 'ru'],
  ['ja', 'ja'],
  ['vi', 'vi'],
  ['en-US', 'en'],
  ['unsupported-language', 'en'],
])(
  'detecting %s initializes the supported language %s',
  async (detected, expected) => {
    localStorage.setItem('i18nextLng', detected)
    const { default: i18n, i18nReady } = await import('../config')
    await i18nReady
    expect(i18n.resolvedLanguage).toBe(expected)
    expect(i18n.hasResourceBundle(expected, 'translation')).toBe(true)
  }
)

test('a failed language download leaves the English fallback usable', async () => {
  vi.doMock('../locales/ru.json', () => {
    throw new Error('Simulated download failure')
  })
  localStorage.setItem('i18nextLng', 'ru')
  const { default: i18n, i18nReady } = await import('../config')
  await i18nReady
  render(
    <I18nextProvider i18n={i18n}>
      <SignInLabel />
    </I18nextProvider>
  )
  expect(
    await screen.findByText(i18n.t('Sign in', { lng: 'en' }))
  ).toBeVisible()
  expect(i18n.resolvedLanguage).toBe('en')
})

test('switching back to a loaded language reuses its resources', async () => {
  const { default: i18n, i18nReady } = await import('../config')
  await i18nReady
  await i18n.changeLanguage('zhCN')
  const { localeBackend } = await import('../locale-backend')
  const read = vi.spyOn(localeBackend, 'read')
  await i18n.changeLanguage('en')
  await i18n.changeLanguage('zhCN')
  expect(i18n.resolvedLanguage).toBe('zhCN')
  expect(read).not.toHaveBeenCalled()
})

function SignInLabel() {
  const { t } = useTranslation()
  return <p>{t('Sign in')}</p>
}

test('a Chinese first visit loads Chinese and the English fallback without other languages', async () => {
  localStorage.setItem('i18nextLng', 'zhCN')
  const { default: i18n, i18nReady } = await import('../config')
  await i18nReady
  render(
    <I18nextProvider i18n={i18n}>
      <SignInLabel />
    </I18nextProvider>
  )
  expect(await screen.findByText('登录')).toBeVisible()
  expect(i18n.hasResourceBundle('en', 'translation')).toBe(true)
  expect(i18n.hasResourceBundle('zhCN', 'translation')).toBe(true)
  for (const language of ['fr', 'ru', 'ja', 'vi', 'zhTW']) {
    expect(i18n.hasResourceBundle(language, 'translation')).toBe(false)
  }
})

test('switching all supported languages updates visible text and preserves the English fallback', async () => {
  const { default: i18n, i18nReady } = await import('../config')
  await i18nReady
  i18n.addResourceBundle('en', 'translation', {
    'Fallback-only test label': 'English fallback',
  })
  render(
    <I18nextProvider i18n={i18n}>
      <SignInLabel />
    </I18nextProvider>
  )
  for (const language of ['zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi', 'en']) {
    await act(() => i18n.changeLanguage(language))
    expect(screen.getByText(i18n.t('Sign in'))).toBeVisible()
    expect(i18n.resolvedLanguage).toBe(language)
    expect(i18n.t('Fallback-only test label')).toBe('English fallback')
  }
  await act(() => i18n.changeLanguage('unsupported-language'))
  expect(i18n.resolvedLanguage).toBe('en')
  expect(screen.getByText(i18n.t('Sign in', { lng: 'en' }))).toBeVisible()
})
