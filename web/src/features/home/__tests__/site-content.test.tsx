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
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, renderHook, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { About } from '@/features/about'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { Home } from '..'

const status = {
  docs_link: 'https://doc.99off.net',
  system_name: 'UniKeyX API',
  HeaderNavModules: JSON.stringify({
    home: true,
    console: true,
    pricing: { enabled: true, requireAuth: false },
    rankings: { enabled: false, requireAuth: false },
    docs: true,
    about: true,
  }),
}
let homeContent = ''

beforeEach(async () => {
  homeContent = ''
  localStorage.clear()
  await i18next.changeLanguage('en')
  useAuthStore.getState().auth.reset()
  useSystemConfigStore.getState().setConfig({
    systemName: 'UniKeyX API',
    logo: '/unikeyx-logo.png',
    footerHtml: '',
  })
  useSystemConfigStore.getState().setLoading(false)
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/home_page_content') {
      return { data: { success: true, data: homeContent } }
    }
    if (url === '/api/about') {
      return {
        data: {
          success: true,
          data: '# UniKeyX.com\n\n先发码，充值成功后再收款\n\nWX：tzy20191024\n\nQQ：2655485',
        },
      }
    }
    if (url === '/api/notice') {
      return { data: { success: true, data: '' } }
    }
    throw new Error(`Unexpected site request: ${url}`)
  })
})

afterEach(() => {
  useAuthStore.getState().auth.reset()
  vi.restoreAllMocks()
})

function siteClient() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(['status'], status)
  return client
}

async function renderSite(path = '/') {
  const root = createRootRoute()
  const home = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: Home,
  })
  const about = createRoute({
    getParentRoute: () => root,
    path: '/about',
    component: About,
  })
  const router = createRouter({
    routeTree: root.addChildren([home, about]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  await router.load()
  return render(
    <QueryClientProvider client={siteClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

it('production navigation hides the image studio and links to the 99Off documentation', () => {
  const client = siteClient()
  function Wrapper(props: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        {props.children}
      </QueryClientProvider>
    )
  }
  const { result } = renderHook(() => useTopNavLinks(), { wrapper: Wrapper })
  expect(result.current).not.toContainEqual(
    expect.objectContaining({ title: 'Professional image generation' })
  )
  expect(result.current).toContainEqual({
    title: 'Docs',
    href: 'https://doc.99off.net',
    external: true,
  })
  expect(result.current).toContainEqual({ title: 'About', href: '/about' })
})

it('the default home shows the production service content with local registration and sign-in routes', async () => {
  await renderSite()
  await screen.findByText(
    'Our mission: reliable models, dependable service, affordable access.'
  )
  const home = within(screen.getByRole('main'))
  expect(
    home.getByRole('heading', { level: 1, name: 'UniKeyX API' })
  ).toBeVisible()
  expect(
    home.getAllByRole('button', { name: 'Register account' })[0]
  ).toHaveAttribute('href', '/sign-up')
  expect(home.getAllByRole('button', { name: 'Sign in' })[0]).toHaveAttribute(
    'href',
    '/sign-in'
  )
  expect(
    home.getByRole('button', { name: 'View documentation' })
  ).toHaveAttribute('href', 'https://doc.99off.net')
  expect(home.getByRole('heading', { name: 'Original models' })).toBeVisible()
  expect(home.getByRole('heading', { name: 'Reliable service' })).toBeVisible()
  expect(home.getByRole('heading', { name: 'Long-term access' })).toBeVisible()
  const image = document.querySelector<HTMLImageElement>('.unikeyx-hero-image')
  expect(image).not.toBeNull()
  expect(image?.src).not.toContain('unsplash.com')
  expect(image).toHaveAttribute('sizes', '100vw')
  expect(image?.srcset).toContain('800w')
  expect(image?.srcset).toContain('2400w')
})

it('an authenticated visitor gets console entries instead of registration', async () => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'local-user', role: 1 })
  await renderSite()
  const entries = await screen.findAllByRole('button', {
    name: 'Enter console',
  })
  expect(entries).toHaveLength(2)
  for (const entry of entries) {
    expect(entry).toHaveAttribute('href', '/dashboard')
  }
  expect(
    screen.queryByRole('button', { name: 'Register account' })
  ).not.toBeInTheDocument()
})

it('the mobile menu reports its open state and restores scrolling when closed', async () => {
  await renderSite()
  await screen.findByRole('heading', { level: 1, name: 'UniKeyX API' })
  const menu = screen.getByRole('button', { name: 'Toggle navigation menu' })
  expect(menu).toHaveAttribute('aria-expanded', 'false')

  await userEvent.click(menu)
  expect(menu).toHaveAttribute('aria-expanded', 'true')
  expect(document.body).toHaveStyle({ overflow: 'hidden' })

  await userEvent.click(menu)
  expect(menu).toHaveAttribute('aria-expanded', 'false')
  expect(document.body.style.overflow).toBe('')
})

it('configured custom home content still takes precedence over the production default', async () => {
  homeContent = '# Custom company home'
  await renderSite()
  expect(
    await screen.findByRole('heading', { name: 'Custom company home' })
  ).toBeVisible()
  expect(
    screen.queryByText(
      'Our mission: reliable models, dependable service, affordable access.'
    )
  ).not.toBeInTheDocument()
})

it('About renders the transferred transaction policy and both production contact details', async () => {
  await renderSite('/about')
  expect(await screen.findByText('先发码，充值成功后再收款')).toBeVisible()
  expect(screen.getByText('WX：tzy20191024')).toBeVisible()
  expect(screen.getByText('QQ：2655485')).toBeVisible()
})
