import { z } from 'zod'

import type { UptimeGroupResult } from '@/features/dashboard/types'
import { api } from '@/lib/api'

const state = z.union([z.literal(-1), z.literal(0), z.literal(1), z.literal(2)])
const slot = z.object({
  start: z.number().finite(),
  end: z.number().finite(),
  status: state,
  uptime: z.number().min(0).max(1).nullable(),
  coverage: z.number().min(0).max(1),
})
const group = z.object({
  slug: z.string(),
  monitorNames: z.array(z.string()),
  status: state,
  uptime: z.number().min(0).max(1).nullable(),
  coverage: z.number().min(0).max(1),
  slots: z.array(slot).length(24),
})
const history = z.object({
  version: z.literal(1),
  generatedAt: z.number().finite(),
  windowStart: z.number().finite(),
  windowEnd: z.number().finite(),
  groups: z.array(group),
})

export type UptimeHistory = z.infer<typeof history>
export type UptimeHistoryGroup = z.infer<typeof group>
export type UptimeHour = z.infer<typeof slot>

export async function getUptimeHistory(
  signal?: AbortSignal
): Promise<UptimeHistory> {
  const response = await api.get<unknown>('/api/uptime/history', {
    signal,
    timeout: 8000,
  })
  return history.parse(response.data)
}

export function findGroupHistory(
  config: UptimeGroupResult,
  data: UptimeHistory | null,
  now = Date.now() / 1000
): UptimeHistoryGroup | null {
  if (!data || now - data.generatedAt > 150 || data.generatedAt > now + 30) {
    return null
  }
  const names = config.monitors.map((monitor) => monitor.name).sort()
  if (!names.length) return null
  return (
    data.groups.find(
      (item) =>
        item.monitorNames.length === names.length &&
        [...item.monitorNames]
          .sort()
          .every((name, index) => name === names[index])
    ) ?? null
  )
}
