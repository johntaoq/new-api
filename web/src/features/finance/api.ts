import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import type { FinanceParams } from './types'

export async function getFinance<T>(
  path: string,
  params: FinanceParams,
  signal?: AbortSignal
): Promise<T> {
  const response = await api.get<{
    success: boolean
    message?: string
    data: T
  }>(`/api/finance/${path}`, {
    params,
    signal,
    // React Query owns cancellation; do not reuse another observer's signal.
    disableDuplicate: Boolean(signal),
  })
  return requireServerSuccess(response.data).data
}

export function useFinanceQuery<T>(
  path: string,
  params: FinanceParams,
  enabled = true,
  keepPreviousPage = false
) {
  const userID = useAuthStore((state) => state.auth.user?.id)
  return useQuery({
    queryKey: ['finance', userID, path, params],
    queryFn: ({ signal }) => getFinance<T>(path, params, signal),
    enabled,
    staleTime: 15_000,
    placeholderData: keepPreviousPage
      ? (previous, previousQuery) => {
          if (
            !previousQuery ||
            previousQuery.queryKey[1] !== userID ||
            previousQuery.queryKey[2] !== path
          ) {
            return undefined
          }
          const previousParams = previousQuery.queryKey[3] as FinanceParams
          const reportKeys = new Set([
            ...Object.keys(previousParams),
            ...Object.keys(params),
          ])
          for (const key of reportKeys) {
            if (
              key !== 'p' &&
              key !== 'page_size' &&
              previousParams[key] !== params[key]
            ) {
              return undefined
            }
          }
          return previous
        }
      : undefined,
  })
}

export async function generateStatements(params: FinanceParams) {
  const response = await api.post<{
    success: boolean
    message?: string
    data: { bill_month: string; generated_user_count: number }
  }>('/api/finance/customer-bills/generate', null, { params })
  return requireServerSuccess(response.data).data
}

export async function exportStatement(
  params: FinanceParams,
  self = false,
  current = false
): Promise<void> {
  let path = self ? 'self/statements/export' : 'customer-bills/export'
  if (current) path = 'self/current/export'
  return downloadFinanceReport(
    path,
    params,
    current
      ? 'current-month-details-CNY.csv'
      : `monthly-bill-${String(params.bill_month)}-CNY.csv`
  )
}

export async function exportCostReport(
  view: 'channels' | 'models',
  params: FinanceParams
): Promise<void> {
  return downloadFinanceReport(
    `channel-cost/${view}/export`,
    params,
    `${view}-costs-${String(params.period)}-USD.csv`
  )
}

async function downloadFinanceReport(
  path: string,
  params: FinanceParams,
  fallbackName: string
): Promise<void> {
  const response = await api.get<Blob>(`/api/finance/${path}`, {
    params,
    responseType: 'blob',
  })
  if (response.data.type.includes('json')) {
    requireServerSuccess(JSON.parse(await response.data.text()))
    throw new Error('Invalid finance report export')
  }
  const url = URL.createObjectURL(response.data)
  const anchor = document.createElement('a')
  anchor.href = url
  const disposition = String(response.headers?.['content-disposition'] ?? '')
  const serverName = disposition.match(/filename="([^"]+)"/)?.[1]
  anchor.download = serverName ?? fallbackName
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  URL.revokeObjectURL(url)
}

export async function setMonthlyRate(billMonth: string, rate: number) {
  const response = await api.put('/api/finance/monthly-billing/rate', {
    bill_month: billMonth,
    rate,
  })
  return requireServerSuccess(response.data)
}
