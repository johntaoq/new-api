import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, beforeAll, describe, expect, test, vi } from 'vitest'

import { getUptimeStatus } from '@/features/dashboard/api'
import {
  getUptimeHistory,
  type UptimeHistory,
} from '@/features/dashboard/lib/uptime-history'

import { UptimePanel } from '../uptime-panel'

vi.mock('@/features/dashboard/api', () => ({ getUptimeStatus: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: { get: vi.fn() } }))
vi.mock('@/features/dashboard/lib/uptime-history', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/features/dashboard/lib/uptime-history')
  >()),
  getUptimeHistory: vi.fn(),
}))

const groups = [
  {
    categoryName: 'Core website',
    monitors: [
      { name: 'homepage', group: 'Internal group', status: 1, uptime: 1 },
      { name: 'api', group: 'Internal group', status: 1, uptime: 1 },
    ],
  },
  {
    categoryName: 'azure GPT',
    monitors: [
      {
        name: 'Azure GPT / gpt-6-luna',
        group: 'Azure GPT',
        status: 1,
        uptime: 1,
      },
    ],
  },
]

function history(status: -1 | 0 | 1 | 2 = 1): UptimeHistory {
  const end = Date.now() / 1000
  return {
    version: 1 as const,
    generatedAt: end,
    windowStart: end - 86400,
    windowEnd: end,
    groups: groups.map((group) => ({
      slug: group.categoryName,
      monitorNames: group.monitors.map((monitor) => monitor.name),
      status,
      uptime: 1,
      coverage: 1,
      slots: Array.from({ length: 24 }, (_, index) => ({
        start: end - (24 - index) * 3600,
        end: end - (23 - index) * 3600,
        status,
        uptime: 1,
        coverage: 1,
      })),
    })),
  }
}

describe('compact uptime panel', () => {
  const animations = Object.getOwnPropertyDescriptor(
    HTMLElement.prototype,
    'getAnimations'
  )
  beforeAll(() => {
    Object.defineProperty(HTMLElement.prototype, 'getAnimations', {
      configurable: true,
      value: () => [],
    })
  })
  afterAll(() => {
    if (animations) {
      Object.defineProperty(HTMLElement.prototype, 'getAnimations', animations)
    } else Reflect.deleteProperty(HTMLElement.prototype, 'getAnimations')
  })

  test('shows one configured name per group, 24 slots and no model or group descriptions', async () => {
    vi.mocked(getUptimeStatus).mockResolvedValue({
      data: groups,
      success: true,
    })
    vi.mocked(getUptimeHistory).mockResolvedValue(history())
    render(<UptimePanel />)
    const core = await screen.findByRole('button', { name: 'Core website' })
    expect(screen.getAllByText('Core website')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'azure GPT' })).toBeVisible()
    expect(within(core).getAllByTestId('uptime-hour')).toHaveLength(24)
    expect(within(core).getByText('100.00%')).toBeVisible()
    expect(screen.queryByText('Internal group')).not.toBeInTheDocument()
    expect(screen.queryByText('Azure GPT / gpt-6-luna')).not.toBeInTheDocument()
    expect(
      screen.queryByText('Grouped monitor status from Uptime Kuma')
    ).not.toBeInTheDocument()
  })

  test('clicking a group opens an accessible expanded dialog and Escape closes it', async () => {
    vi.mocked(getUptimeStatus).mockResolvedValue({
      data: groups,
      success: true,
    })
    vi.mocked(getUptimeHistory).mockResolvedValue(history())
    render(<UptimePanel />)
    await userEvent.click(
      await screen.findByRole('button', { name: 'azure GPT' })
    )
    const dialog = await screen.findByRole('dialog', { name: 'azure GPT' })
    expect(within(dialog).getAllByTestId('uptime-hour')).toHaveLength(24)
    expect(
      within(dialog).getAllByRole('button', { name: /\d{4}-\d{2}-\d{2}/ })
        .length
    ).toBe(24)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  test('failed history requests keep names but show unknown rather than false red or green', async () => {
    vi.mocked(getUptimeStatus).mockResolvedValue({
      data: groups,
      success: true,
    })
    vi.mocked(getUptimeHistory).mockRejectedValue(new Error('Unavailable'))
    render(<UptimePanel />)
    const core = await screen.findByRole('button', { name: 'Core website' })
    expect(within(core).getByText('--')).toBeVisible()
    expect(
      within(core).getByRole('img', { name: 'No data available' })
    ).toHaveAttribute('data-status', '-1')
  })

  test('pending stays yellow and confirmed failure stays red', async () => {
    vi.mocked(getUptimeStatus).mockResolvedValue({
      data: groups,
      success: true,
    })
    const data = history(2)
    data.groups[1].status = 0
    vi.mocked(getUptimeHistory).mockResolvedValue(data)
    render(<UptimePanel />)
    const core = await screen.findByRole('button', { name: 'Core website' })
    expect(within(core).getByRole('img', { name: 'Pending' })).toHaveAttribute(
      'data-status',
      '2'
    )
    expect(
      within(screen.getByRole('button', { name: 'azure GPT' })).getByRole(
        'img',
        { name: 'Failed' }
      )
    ).toHaveAttribute('data-status', '0')
  })

  test('refresh replaces stale healthy data with unknown when history is too old', async () => {
    vi.mocked(getUptimeStatus).mockResolvedValue({
      data: groups,
      success: true,
    })
    const data = history()
    data.generatedAt -= 300
    vi.mocked(getUptimeHistory).mockResolvedValue(data)
    render(<UptimePanel />)
    const core = await screen.findByRole('button', { name: 'Core website' })
    expect(within(core).getByText('--')).toBeVisible()
    expect(
      within(core).getByRole('img', { name: 'No data available' })
    ).toBeVisible()
  })
})
