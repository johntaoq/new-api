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
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { TopNav } from '@/components/layout/components/top-nav'
import type { TopNavLink } from '@/components/layout/types'
import { api } from '@/lib/api'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

it('replaces the image studio link with one authorized Pic365 action between models and docs', async () => {
  const client = new QueryClient()
  client.setQueryData(['pic365-entry'], {
    enabled: true,
    start_url: 'https://www.pic365.org/api/auth/newapi40/start',
  })
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true } })
  const links = [
    { title: 'Model Square', href: '/pricing' },
    {
      title: 'Pic365',
      href: 'https://www.pic365.org/',
      external: true,
      pic365: true,
    },
    { title: 'Docs', href: 'https://doc.unikeyx.com', external: true },
  ] as TopNavLink[]
  const root = createRootRoute({ component: () => <TopNav links={links} /> })
  const router = createRouter({
    routeTree: root,
    history: createMemoryHistory(),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  const action = await screen.findByRole('button', { name: 'Pic365' })
  expect(action.previousElementSibling).toHaveTextContent('Model Square')
  expect(action.nextElementSibling).toHaveTextContent('Docs')
  expect(screen.queryByRole('link', { name: 'Pic365' })).not.toBeInTheDocument()
  expect(
    screen.queryByText('Professional image generation')
  ).not.toBeInTheDocument()
  const popup = {
    closed: false,
    focus: vi.fn(),
    location: { href: 'about:blank', replace: vi.fn() },
  }
  const open = vi
    .spyOn(window, 'open')
    .mockReturnValue(popup as unknown as Window)
  await userEvent.setup().click(action)
  expect(open).toHaveBeenCalledWith('', 'newapi-pic365')
  expect(popup.location.replace).toHaveBeenCalledWith(
    'https://www.pic365.org/api/auth/newapi40/start'
  )
  popup.closed = true
  client.clear()
})
