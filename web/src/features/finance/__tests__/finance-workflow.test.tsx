import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import i18next from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { exportStatement } from '../api'
import { FinanceCustomers } from '../components/customers'
import { FinanceRemoteTable } from '../components/shared'
import { WalletMonthlyStatements } from '../components/wallet-statements'
import { useFinanceFormat } from '../use-finance-format'

const i18n = i18next.createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: 'en',
  resources: { en: { translation: {} } },
})
let client: QueryClient

beforeEach(() => {
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

function financeTree(content: React.ReactNode) {
  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={client}>{content}</QueryClientProvider>
    </I18nextProvider>
  )
}

describe('finance reports', () => {
  it('shows current details in CNY and downloads only the signed-in user export', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [{ id: 1, amount_cny: -14, entry_type: 'consume' }],
          total: 1,
        },
      },
    })
    render(financeTree(<WalletMonthlyStatements />))
    fireEvent.click(screen.getByRole('button', { name: 'Monthly statements' }))
    fireEvent.click(
      await screen.findByRole('tab', { name: 'Current month details' })
    )
    expect(await screen.findByText('Balance change (CNY)')).toBeVisible()
    expect(screen.queryByText('COS')).not.toBeInTheDocument()
    expect(get).toHaveBeenCalledWith(
      '/api/finance/self/current',
      expect.anything()
    )
    const blob = new Blob(['Balance change (CNY)\n-14'], { type: 'text/csv' })
    get.mockResolvedValueOnce({
      data: blob,
      headers: {
        'content-disposition': 'attachment; filename="current-details-CNY.csv"',
      },
    })
    const create = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue('blob:local-test')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined)
    await exportStatement({}, true, true)
    expect(get).toHaveBeenLastCalledWith('/api/finance/self/current/export', {
      params: {},
      responseType: 'blob',
    })
    expect(create).toHaveBeenCalledWith(blob)
    expect(click).toHaveBeenCalledOnce()
  })
  it('shows loading before the requested page and keeps long customer names readable', async () => {
    const name = 'Local finance customer with a deliberately long account name'
    let resolve!: (value: unknown) => void
    vi.spyOn(api, 'get').mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }) as ReturnType<typeof api.get>
    )
    render(
      financeTree(
        <FinanceRemoteTable
          title='Customers'
          path='customer-bills/summary'
          params={{ bill_month: '2026-10' }}
          columns={[{ key: 'username', label: 'Customer' }]}
        />
      )
    )
    expect(screen.queryByText(name)).not.toBeInTheDocument()
    resolve({
      data: {
        success: true,
        data: { items: [{ username: name }], page: 1, page_size: 10, total: 1 },
      },
    })
    expect(await screen.findByText(name)).toBeVisible()
    expect(api.get).toHaveBeenCalledWith(
      '/api/finance/customer-bills/summary',
      expect.objectContaining({
        params: { bill_month: '2026-10', p: 1, page_size: 10 },
      })
    )
  })

  it('shows an empty state for an empty report and a retry control on failure', async () => {
    vi.spyOn(api, 'get')
      .mockResolvedValueOnce({
        data: { success: true, data: { items: [], total: 0 } },
      })
      .mockRejectedValueOnce(new Error('local test unavailable'))
    const view = render(
      financeTree(
        <FinanceRemoteTable
          title='Customers'
          path='empty'
          params={{}}
          columns={[]}
        />
      )
    )
    expect(await screen.findByText('No data')).toBeVisible()
    view.rerender(
      financeTree(
        <FinanceRemoteTable
          key='error'
          title='Customers'
          path='error'
          params={{}}
          columns={[]}
        />
      )
    )
    expect(
      await screen.findByRole('button', { name: /retry|try again/i })
    ).toBeEnabled()
  })

  it('keeps statements read-only and allows only operators to schedule next month rates', async () => {
    vi.spyOn(api, 'get').mockImplementation(async (url) => ({
      data: {
        success: true,
        data: String(url).endsWith('monthly-billing')
          ? {
              current_month: '2026-10',
              next_month: '2026-11',
              months: [
                {
                  bill_month: '2026-10',
                  cny_rate: 7,
                  closes_at: 1793464200,
                  finalized_at: 0,
                  last_error: '',
                },
              ],
            }
          : { items: [], total: 0 },
      },
    }))
    const post = vi.spyOn(api, 'post')
    const put = vi.spyOn(api, 'put').mockResolvedValue({
      data: { success: true, data: null },
    })
    useAuthStore.getState().auth.setUser({
      id: 2,
      username: 'reader',
      role: 10,
      permissions: { admin_permissions: { finance: { read: true } } },
    })
    const view = render(financeTree(<FinanceCustomers unit='usd' />))
    expect(
      screen.queryByRole('button', { name: 'Generate statements' })
    ).not.toBeInTheDocument()
    await screen.findByText(/2026-10/)
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    view.unmount()
    useAuthStore
      .getState()
      .auth.setUser({ id: 1, username: 'operator', role: 100 })
    render(financeTree(<FinanceCustomers unit='usd' />))
    const input = await screen.findByLabelText(/Next month exchange rate/)
    fireEvent.change(input, { target: { value: '7.3' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(put).not.toHaveBeenCalled()
    expect(post).not.toHaveBeenCalled()
    fireEvent.click(await screen.findByRole('button', { name: 'Confirm' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith('/api/finance/monthly-billing/rate', {
        bill_month: '2026-11',
        rate: 7.3,
      })
    )
  })

  it('shows stored CNY amounts in the wallet and never calls administrator APIs', async () => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          items: [
            {
              bill_month: '2026-09',
              user_id: 1,
              username: 'customer',
              cny_rate: 7,
              total_consume_usd: 10,
              total_consume_cny: 70,
              total_consume_cos: 10000,
              paid_consume_usd: 10,
              paid_consume_cny: 70,
              paid_consume_cos: 10000,
              gift_consume_usd: 0,
              gift_consume_cny: 0,
              gift_consume_cos: 0,
            },
          ],
          total: 1,
        },
      },
    })
    const post = vi.spyOn(api, 'post')
    render(financeTree(<FinanceCustomers self />))
    expect((await screen.findAllByText(/70\.00/)).length).toBeGreaterThan(0)
    expect(screen.queryByLabelText('Customer search')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    expect(
      get.mock.calls.every(
        ([path]) => String(path) === '/api/finance/self/statements'
      )
    ).toBe(true)
    expect(post).not.toHaveBeenCalled()
  })

  it.each(['zhCN', 'zhTW', 'en', 'fr', 'ru', 'ja', 'vi', 'invalid_locale'])(
    'formats fixed USD, CNY and COS amounts for %s without a locale crash',
    async (language) => {
      await i18n.changeLanguage(language)
      function Amounts() {
        const format = useFinanceFormat()
        return (
          <div>
            {format.amount(1234.56789)} / {format.amount(1234.56789, 'cos')} /{' '}
            {format.amount(70, 'cny')}
          </div>
        )
      }
      render(financeTree(<Amounts />))
      expect(screen.getByText(/COS/)).toBeVisible()
      expect(screen.queryByText(/NaN/)).not.toBeInTheDocument()
      await i18n.changeLanguage('en')
    }
  )
})
