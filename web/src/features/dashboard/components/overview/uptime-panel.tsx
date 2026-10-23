/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Activity, RotateCw } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { getUptimeStatus } from '@/features/dashboard/api'
import {
  findGroupHistory,
  getUptimeHistory,
  type UptimeHistory,
  type UptimeHour,
} from '@/features/dashboard/lib/uptime-history'
import type { UptimeGroupResult } from '@/features/dashboard/types'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { PanelWrapper } from '../ui/panel-wrapper'

const STATUS_COLOR_MAP: Record<number, string> = {
  1: 'bg-emerald-500',
  0: 'bg-red-500',
  2: 'bg-amber-500',
}
const DEFAULT_STATUS_COLOR = 'bg-muted-foreground/40'

function StatusDot(props: { status: number }) {
  const { t } = useTranslation()
  const labels: Record<number, string> = {
    1: t('Available'),
    0: t('Failed'),
    2: t('Pending'),
  }
  return (
    <span
      role='img'
      aria-label={labels[props.status] ?? t('No data available')}
      title={labels[props.status] ?? t('No data available')}
      data-status={props.status}
      className={cn(
        'inline-block size-2.5 shrink-0 rounded-full',
        STATUS_COLOR_MAP[props.status] ?? DEFAULT_STATUS_COLOR
      )}
    />
  )
}

function hourLabel(hour: UptimeHour) {
  return `${formatTimestampToDate(hour.start)} - ${formatTimestampToDate(hour.end)}`
}

export function UptimePanel() {
  const { t } = useTranslation()
  const [groups, setGroups] = useState<UptimeGroupResult[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [history, setHistory] = useState<UptimeHistory | null>(null)
  const [selected, setSelected] = useState<string | null>(null)
  const [selectedHour, setSelectedHour] = useState<number | null>(null)
  const request = useRef<AbortController | null>(null)

  const refresh = useCallback(async () => {
    request.current?.abort()
    const abortController = new AbortController()
    request.current = abortController
    setRefreshing(true)
    const [statusResult, historyResult] = await Promise.allSettled([
      getUptimeStatus(),
      getUptimeHistory(abortController.signal),
    ])
    if (abortController.signal.aborted) return
    if (statusResult.status === 'fulfilled') {
      setGroups(statusResult.value?.data || [])
    }
    setHistory(
      historyResult.status === 'fulfilled' ? historyResult.value : null
    )
    setLoading(false)
    setRefreshing(false)
  }, [])

  useEffect(() => {
    void refresh()
    const timer = window.setInterval(() => {
      void refresh()
    }, 60_000)
    return () => {
      request.current?.abort()
      window.clearInterval(timer)
    }
  }, [refresh])

  const selectedGroup = groups.find((group) => group.categoryName === selected)
  const expanded = selectedGroup
    ? findGroupHistory(selectedGroup, history)
    : null
  const selectedSlot =
    selectedHour === null ? null : expanded?.slots[selectedHour]
  const labels: Record<number, string> = {
    1: t('Available'),
    0: t('Failed'),
    2: t('Pending'),
  }

  return (
    <>
      <PanelWrapper
        title={
          <span className='flex items-center gap-2'>
            <IconBadge tone='success' size='sm'>
              <Activity />
            </IconBadge>
            {t('Uptime')}{' '}
            <span className='text-muted-foreground text-xs font-normal'>
              24h
            </span>
          </span>
        }
        loading={loading}
        empty={!groups.length}
        emptyMessage={t('No uptime monitoring configured')}
        height='h-64'
        contentClassName='p-0'
        headerActions={
          <Button
            variant='ghost'
            size='sm'
            onClick={() => {
              void refresh()
            }}
            disabled={refreshing}
            className='size-7 p-0'
            aria-label={t('Refresh')}
            title={t('Refresh')}
          >
            <RotateCw
              className={cn('size-3.5', refreshing && 'animate-spin')}
              aria-hidden='true'
            />
          </Button>
        }
      >
        <div className='max-h-96 divide-y overflow-y-auto'>
          {groups.map((group) => {
            const data = findGroupHistory(group, history)
            return (
              <Button
                key={group.categoryName}
                variant='ghost'
                aria-label={group.categoryName}
                aria-haspopup='dialog'
                onClick={() => {
                  setSelected(group.categoryName)
                  setSelectedHour(null)
                }}
                className='flex h-auto min-h-20 w-full flex-col items-stretch gap-3 rounded-none px-4 py-4 text-left sm:px-5'
              >
                <span className='grid min-w-0 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3'>
                  <span className='min-w-0 text-sm font-medium [overflow-wrap:anywhere] break-words whitespace-normal'>
                    {group.categoryName}
                  </span>
                  <span className='shrink-0 font-mono text-sm tabular-nums'>
                    {data?.uptime == null
                      ? '--'
                      : `${(data.uptime * 100).toFixed(2)}%`}
                  </span>
                  <StatusDot status={data?.status ?? -1} />
                </span>
                <span className='grid w-full min-w-0 grid-cols-[repeat(24,minmax(0,1fr))] gap-1'>
                  {Array.from({ length: 24 }, (_, index) => {
                    const hour = data?.slots[index]
                    const status = hour?.status ?? -1
                    return (
                      <span
                        key={index}
                        data-testid='uptime-hour'
                        data-status={status}
                        title={
                          hour
                            ? `${hourLabel(hour)}: ${labels[status] ?? t('No data available')}`
                            : t('No data available')
                        }
                        className={cn(
                          'block h-3 min-w-0 rounded-[2px]',
                          STATUS_COLOR_MAP[status] ?? DEFAULT_STATUS_COLOR
                        )}
                      />
                    )
                  })}
                </span>
              </Button>
            )
          })}
        </div>
      </PanelWrapper>
      <Dialog
        open={selected !== null}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
        title={selected ?? t('Uptime')}
        contentClassName='w-[calc(100vw-2rem)] sm:max-w-5xl'
      >
        <div className='flex items-center justify-between gap-4 border-b pb-4'>
          <span className='text-muted-foreground text-sm'>24h</span>
          <span className='flex items-center gap-3 font-mono text-base tabular-nums'>
            {expanded?.uptime == null
              ? '--'
              : `${(expanded.uptime * 100).toFixed(2)}%`}
            <StatusDot status={expanded?.status ?? -1} />
          </span>
        </div>
        <div className='grid grid-cols-6 gap-2 py-5 sm:grid-cols-12 lg:grid-cols-24'>
          {Array.from({ length: 24 }, (_, index) => {
            const hour = expanded?.slots[index]
            const status = hour?.status ?? -1
            return (
              <Button
                key={index}
                data-testid='uptime-hour'
                data-status={status}
                variant='ghost'
                disabled={!hour}
                aria-label={hour ? hourLabel(hour) : t('No data available')}
                aria-pressed={selectedHour === index}
                title={hour ? hourLabel(hour) : t('No data available')}
                onClick={() => setSelectedHour(index)}
                className={cn(
                  'h-auto min-w-0 flex-col gap-2 px-1 py-2',
                  selectedHour === index && 'ring-primary ring-2'
                )}
              >
                <span
                  className={cn(
                    'block h-10 w-full rounded-[3px]',
                    STATUS_COLOR_MAP[status] ?? DEFAULT_STATUS_COLOR
                  )}
                />
                <span className='text-muted-foreground text-[11px] tabular-nums'>
                  {hour
                    ? formatTimestampToDate(hour.start).slice(11, 16)
                    : '--'}
                </span>
              </Button>
            )
          })}
        </div>
        <div className='flex flex-wrap items-center gap-x-5 gap-y-2 border-t pt-4 text-xs'>
          {[1, 2, 0, -1].map((status) => (
            <span key={status} className='inline-flex items-center gap-2'>
              <StatusDot status={status} />
              {labels[status] ?? t('No data available')}
            </span>
          ))}
        </div>
        {selectedSlot && (
          <div className='mt-4 flex flex-wrap items-center justify-between gap-3 border-t pt-4 text-sm'>
            <span className='tabular-nums'>{hourLabel(selectedSlot)}</span>
            <span className='flex items-center gap-2'>
              <StatusDot status={selectedSlot.status} />
              {labels[selectedSlot.status] ?? t('No data available')}
            </span>
          </div>
        )}
      </Dialog>
    </>
  )
}
