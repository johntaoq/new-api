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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  cleanup,
  render,
  renderHook,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { StrictMode } from 'react'
import { toast } from 'sonner'
import { afterEach, expect, it, vi } from 'vitest'

import { SidebarProvider } from '@/components/ui/sidebar'
import { useSidebarData } from '@/hooks/use-sidebar-data'
import { api } from '@/lib/api'

import { Pic365Authorize, Pic365Launch } from '..'

const windows: { closed: boolean }[] = []
const clients: QueryClient[] = []

afterEach(() => {
  cleanup()
  for (const popup of windows) popup.closed = true
  windows.length = 0
  for (const client of clients) client.clear()
  clients.length = 0
  vi.restoreAllMocks()
})

function launchFixture(enabled = true) {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        enabled,
        start_url: 'https://www.pic365.org/api/auth/newapi16/start',
      },
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  render(
    <QueryClientProvider client={client}>
      <Pic365Launch />
      <SidebarProvider>
        <Pic365Launch presentation='sidebar' />
      </SidebarProvider>
    </QueryClientProvider>
  )
}

function popupFixture() {
  const popup = {
    closed: false,
    focus: vi.fn(),
    location: { href: 'about:blank', replace: vi.fn() },
  }
  windows.push(popup)
  return popup
}

it('does not repeat one-time authorization when StrictMode replays effects', async () => {
  const post = vi.spyOn(api, 'post').mockRejectedValue(new Error('unavailable'))
  render(
    <StrictMode>
      <Pic365Authorize state={'s'.repeat(43)} challenge={'c'.repeat(43)} />
    </StrictMode>
  )
  await screen.findByText('Pic365 authorization is unavailable')
  expect(post).toHaveBeenCalledTimes(1)
  expect(post).toHaveBeenCalledWith(
    '/api/user/self/pic365/authorize',
    { state: 's'.repeat(43), code_challenge: 'c'.repeat(43) },
    { singleUseAuthorization: true }
  )
})

it('shows the launch action only when integration is configured', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        enabled: true,
        start_url: 'https://www.pic365.org/api/auth/newapi16/start',
      },
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Pic365Launch />
    </QueryClientProvider>
  )
  expect(await screen.findByRole('button', { name: 'Pic365' })).toBeVisible()
  client.clear()
})

it('places Pic365 image generation immediately after Playground without duplicating it in General', () => {
  const { result } = renderHook(useSidebarData)
  const items =
    result.current.navGroups.find((group) => group.id === 'chat')?.items ??
    []
  const index = items.findIndex((item) => item.url === '/playground')
  expect(index).toBeGreaterThanOrEqual(0)
  expect(items[index + 1]).toMatchObject({
    title: 'Pic365 Image Generation',
    type: 'pic365',
    url: '/pic365',
  })
  expect(
    result.current.navGroups
      .find((group) => group.id === 'general')
      ?.items.some((item) => item.type === 'pic365')
  ).toBe(false)
})

it('shows a red filled flame on the sidebar entry while retaining its accessible name', async () => {
  launchFixture()
  const button = await screen.findByRole('button', {
    name: 'Pic365 Image Generation',
  })
  const icon = button.querySelector('svg')
  expect(icon).toHaveClass('lucide-flame', 'fill-red-500', 'text-red-600')
  expect(icon).toHaveAttribute('aria-hidden', 'true')
})

it('top and sidebar entries open one new tab and reuse it without navigating the console or reloading Pic365', async () => {
  const user = userEvent.setup()
  const popup = popupFixture()
  const open = vi
    .spyOn(window, 'open')
    .mockReturnValue(popup as unknown as Window)
  const originalURL = window.location.href
  launchFixture()
  await user.click(await screen.findByRole('button', { name: 'Pic365' }))
  await user.click(
    await screen.findByRole('button', { name: 'Pic365 Image Generation' })
  )
  expect(open).toHaveBeenCalledTimes(1)
  expect(open).toHaveBeenCalledWith('', 'newapi-pic365')
  expect(popup.location.replace).toHaveBeenCalledExactlyOnceWith(
    'https://www.pic365.org/api/auth/newapi16/start'
  )
  expect(popup.focus).toHaveBeenCalledTimes(2)
  expect(window.location.href).toBe(originalURL)
})

it('reopens Pic365 when the previously opened tab has been closed', async () => {
  const user = userEvent.setup()
  const first = popupFixture()
  const second = popupFixture()
  const open = vi
    .spyOn(window, 'open')
    .mockReturnValueOnce(first as unknown as Window)
    .mockReturnValueOnce(second as unknown as Window)
  launchFixture()
  const button = await screen.findByRole('button', { name: 'Pic365' })
  await user.click(button)
  first.closed = true
  await user.click(button)
  expect(open).toHaveBeenCalledTimes(2)
  expect(second.location.replace).toHaveBeenCalledExactlyOnceWith(
    'https://www.pic365.org/api/auth/newapi16/start'
  )
})

it('reattaches to a named cross-origin Pic365 tab without restarting authorization', async () => {
  const user = userEvent.setup()
  const existing = {
    closed: false,
    focus: vi.fn(),
    get location(): Location {
      throw new DOMException('cross-origin', 'SecurityError')
    },
  }
  windows.push(existing)
  vi.spyOn(window, 'open').mockReturnValue(existing as unknown as Window)
  launchFixture()
  await user.click(await screen.findByRole('button', { name: 'Pic365' }))
  expect(existing.focus).toHaveBeenCalledOnce()
})

it('reports blocked pop-ups and leaves the console in place', async () => {
  const user = userEvent.setup()
  const error = vi.spyOn(toast, 'error').mockImplementation(() => 'test-toast')
  vi.spyOn(window, 'open').mockReturnValue(null)
  const originalURL = window.location.href
  launchFixture()
  await user.click(await screen.findByRole('button', { name: 'Pic365' }))
  expect(error).toHaveBeenCalledWith('Pic365 pop-up was blocked')
  expect(window.location.href).toBe(originalURL)
})

it('hides both Pic365 entries when the integration is disabled', async () => {
  launchFixture(false)
  await waitFor(() =>
    expect(clients[0].getQueryState(['pic365-entry'])?.status).toBe('success')
  )
  expect(
    screen.queryByRole('button', { name: /Pic365/ })
  ).not.toBeInTheDocument()
})
