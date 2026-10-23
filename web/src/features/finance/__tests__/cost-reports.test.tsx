import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { exportCostReport, getFinance } from '../api'
import { FinanceChannels } from '../components/channels'
import { FinanceRemoteTable } from '../components/shared'
import { useFinanceFormat } from '../use-finance-format'

const i18n = i18next.createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: 'en',
  resources: { en: { translation: {} } },
})
let client: QueryClient
beforeEach(async () => {
  await i18n.changeLanguage('en')
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'local-admin', role: 100 })
})
afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.getState().auth.reset()
  vi.restoreAllMocks()
})
function reports(content: React.ReactNode) {
  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>{content}</QueryClientProvider>
    </I18nextProvider>
  )
}

describe('cost report pagination', () => {
  it('does not reuse a canceled request when the report is immediately reopened', async () => {
    const adapter = api.defaults.adapter
    let release!: () => void
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const request = vi.fn(async (config) => {
      await gate
      return {
        data: { success: true, data: { total: 1 } },
        status: 200,
        statusText: 'OK',
        headers: {},
        config,
      }
    })
    api.defaults.adapter = request
    try {
      const canceled = new AbortController()
      const first = getFinance(
        'channel-cost/cancel-test',
        {},
        canceled.signal
      ).catch((error: unknown) => error)
      await waitFor(() => expect(request).toHaveBeenCalledOnce())
      canceled.abort()
      const reopened = getFinance(
        'channel-cost/cancel-test',
        {},
        new AbortController().signal
      )
      release()
      await first
      await expect(reopened).resolves.toEqual({ total: 1 })
    } finally {
      release()
      api.defaults.adapter = adapter
    }
  })

  it('keeps the current report and pagination available after a failed page request', async () => {
    const user = userEvent.setup()
    vi.spyOn(api, 'get')
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { items: [{ name: 'First channel' }], total: 30 },
        },
      })
      .mockRejectedValueOnce(new Error('Page unavailable'))
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { items: [{ name: 'Second channel' }], total: 30 },
        },
      })
    render(
      reports(
        <FinanceRemoteTable
          title='Channel costs'
          path='channel-cost/channels'
          params={{ period: '2026-10' }}
          columns={[{ key: 'name', label: 'Channel' }]}
        />
      )
    )
    await screen.findByText('First channel')
    const next = screen.getByRole('button', { name: 'Go to next page' })
    await user.click(next)
    const retry = await screen.findByRole('button', {
      name: /retry|try again/i,
    })
    expect(next).toHaveFocus()
    expect(screen.getByText('First channel')).toBeVisible()
    await user.click(retry)
    expect(await screen.findByText('Second channel')).toBeVisible()
  })

  it('does not show the previous report while a different period loads', async () => {
    vi.spyOn(api, 'get')
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { items: [{ name: 'October channel' }], total: 1 },
        },
      })
      .mockImplementationOnce(() => new Promise(() => undefined))
    const view = render(
      reports(
        <FinanceRemoteTable
          title='Channel costs'
          path='channel-cost/channels'
          params={{ period: '2026-10' }}
          columns={[{ key: 'name', label: 'Channel' }]}
        />
      )
    )
    await screen.findByText('October channel')
    view.rerender(
      reports(
        <FinanceRemoteTable
          title='Channel costs'
          path='channel-cost/channels'
          params={{ period: '2026-11' }}
          columns={[{ key: 'name', label: 'Channel' }]}
        />
      )
    )
    expect(screen.queryByText('October channel')).not.toBeInTheDocument()
  })

  it('keeps the focused page button and previous rows while fetching only that report', async () => {
    const user = userEvent.setup()
    let resolve!: (value: unknown) => void
    const get = vi.spyOn(api, 'get').mockImplementation((path, options) => {
      if (String(path).endsWith('/channels') && options?.params.p === 2) {
        return new Promise((done) => {
          resolve = done
        }) as ReturnType<typeof api.get>
      }
      return Promise.resolve({
        data: {
          success: true,
          data: {
            items: [
              {
                name: String(path).endsWith('/models')
                  ? 'Shared model'
                  : 'First channel',
              },
            ],
            total: 30,
          },
        },
      }) as ReturnType<typeof api.get>
    })
    render(
      reports(
        <>
          <FinanceRemoteTable
            title='Channel costs'
            path='channel-cost/channels'
            params={{ period: '2026-10' }}
            columns={[{ key: 'name', label: 'Channel' }]}
          />
          <FinanceRemoteTable
            title='Model costs'
            path='channel-cost/models'
            params={{ period: '2026-10' }}
            columns={[{ key: 'name', label: 'Model' }]}
          />
        </>
      )
    )
    await screen.findByText('First channel')
    await screen.findByText('Shared model')
    const section = screen.getByRole('region', { name: 'Channel costs' })
    const next = within(section).getByRole('button', {
      name: 'Go to next page',
    })
    await user.click(next)
    await waitFor(() => expect(resolve).toBeTypeOf('function'))
    expect(next).toHaveFocus()
    expect(next).toBeInTheDocument()
    expect(screen.getByText('First channel')).toBeVisible()
    expect(
      get.mock.calls.filter(([path]) => String(path).endsWith('/models'))
    ).toHaveLength(1)
    await act(async () =>
      resolve({
        data: {
          success: true,
          data: { items: [{ name: 'Second channel' }], total: 30 },
        },
      })
    )
    expect(await screen.findByText('Second channel')).toBeVisible()
    expect(next).toHaveFocus()
    expect(
      within(section).getByRole('button', { name: 'Go to next page' })
    ).toBe(next)
  })
})

describe('cost report exports', () => {
  it('rejects a server error without downloading it as a CSV file', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: new Blob(['{"success":false,"message":"Permission denied"}'], {
        type: 'application/json',
      }),
    })
    const create = vi.spyOn(URL, 'createObjectURL')
    await expect(
      exportCostReport('models', { period: '2026-10' })
    ).rejects.toThrow('Permission denied')
    expect(create).not.toHaveBeenCalled()
  })
  it.each(['Channel costs', 'Model costs'])(
    'downloads the complete %s report for the selected period without refreshing tables',
    async (title) => {
      const user = userEvent.setup()
      const blob = new Blob(['\ufeff模型,成本（USD）\r\nshared,12.35'], {
        type: 'text/csv',
      })
      const get = vi.spyOn(api, 'get').mockImplementation((path) => {
        if (String(path).endsWith('/export')) {
          return Promise.resolve({
            data: blob,
            headers: {
              'content-disposition': 'attachment; filename="costs.csv"',
            },
          }) as ReturnType<typeof api.get>
        }
        return Promise.resolve({
          data: {
            success: true,
            data: String(path).endsWith('/summary')
              ? {
                  kpis: { total_cost_usd: 12.345678, total_tokens: 100 },
                  charts: {},
                }
              : {
                  items: [
                    {
                      model_name: 'Shared model',
                      channel_name: 'Local channel',
                      cost_usd: 12.345678,
                    },
                  ],
                  total: 1,
                },
          },
        }) as ReturnType<typeof api.get>
      })
      const create = vi
        .spyOn(URL, 'createObjectURL')
        .mockReturnValue('blob:costs')
      vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
      const click = vi
        .spyOn(HTMLAnchorElement.prototype, 'click')
        .mockImplementation(() => undefined)
      render(
        reports(
          <FinanceChannels params={{ period_type: 'year', period: '2026' }} />
        )
      )
      await screen.findByText('Shared model')
      const section = screen.getByRole('region', { name: title })
      if (title === 'Model costs') {
        expect(
          within(section).queryByRole('columnheader', { name: 'Channel' })
        ).not.toBeInTheDocument()
      }
      await user.click(
        within(section).getByRole('button', { name: 'Export CSV' })
      )
      await waitFor(() => expect(click).toHaveBeenCalledOnce())
      expect(create).toHaveBeenCalledWith(blob)
      expect(get).toHaveBeenLastCalledWith(
        `/api/finance/channel-cost/${title === 'Model costs' ? 'models' : 'channels'}/export`,
        {
          params: { period_type: 'year', period: '2026' },
          responseType: 'blob',
        }
      )
      expect(
        get.mock.calls.filter(([path]) => !String(path).endsWith('/export'))
      ).toHaveLength(3)
    }
  )
})

describe('finance number precision', () => {
  it.each(['zhCN', 'zhTW', 'en', 'fr', 'ru', 'ja', 'vi', 'invalid_locale'])(
    'displays exactly two fractional digits for %s and updates on language change',
    async (language) => {
      await i18n.changeLanguage(language)
      function Numbers() {
        const format = useFinanceFormat()
        return (
          <output aria-label='Finance amounts'>
            {format.amount(1234.56789)} | {format.amount(1234.56789, 'cny')} |{' '}
            {format.amount(1234.56789, 'cos')} | {format.number(1234.56789)} |{' '}
            {format.number(100)}
          </output>
        )
      }
      render(reports(<Numbers />))
      const output = screen.getByLabelText('Finance amounts')
      const fractions = [
        ...(output.textContent ?? '').matchAll(/[.,](\d+)(?!\d)/g),
      ]
        .map((match) => match[1])
        .filter((part) => part.length !== 3)
      expect(fractions).toEqual(['57', '57', '57', '57', '00'])
      await act(async () => {
        await i18n.changeLanguage('en')
      })
      expect(output).toHaveTextContent(
        '$1,234.57 | CN¥1,234.57 | 1,234.57 COS | 1,234.57 | 100.00'
      )
    }
  )
})
