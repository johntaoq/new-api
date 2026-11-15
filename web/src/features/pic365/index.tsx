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
import { useQuery } from '@tanstack/react-query'
import { Flame, Images } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { api } from '@/lib/api'
import { handleServerError } from '@/lib/handle-server-error'
import { requireServerSuccess } from '@/lib/server-error-message'

let pic365Window: Window | null = null

function openPic365Window(startURL: string): boolean {
  if (pic365Window && !pic365Window.closed) {
    pic365Window.focus()
    return true
  }
  // Resolve the named tab without navigating it, preserving ongoing work.
  const popup = window.open('', 'newapi-pic365')
  if (!popup) return false
  pic365Window = popup
  let isBlank = false
  try {
    isBlank = popup.location.href === 'about:blank'
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'SecurityError')) {
      throw error
    }
    // An existing cross-origin Pic365 tab must not restart its SSO flow.
  }
  if (isBlank) popup.location.replace(startURL)
  popup.focus()
  return true
}

function Pic365SidebarLaunch({ onClick }: { onClick: () => void }) {
  const { t } = useTranslation()
  const { setOpenMobile } = useSidebar()
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={t('Pic365 Image Generation')}
        onClick={() => {
          onClick()
          setOpenMobile(false)
        }}
      >
        <Flame
          aria-hidden='true'
          className='shrink-0 fill-red-500 text-red-600 dark:text-red-400'
        />
        <span className='min-w-0 flex-1 truncate'>
          {t('Pic365 Image Generation')}
        </span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

export function Pic365Launch({
  presentation = 'header',
  className,
  onClick,
}: {
  presentation?: 'header' | 'sidebar' | 'navigation'
  className?: string
  onClick?: () => void
}) {
  const { t } = useTranslation()
  const entry = useQuery({
    queryKey: ['pic365-entry'],
    queryFn: async () => {
      const response = await api.get('/api/user/self/pic365')
      return requireServerSuccess(response.data).data as {
        enabled: boolean
        start_url?: string
      }
    },
    staleTime: 300_000,
    retry: false,
  })
  if (!entry.data?.enabled || !entry.data.start_url) return null
  const startURL = entry.data.start_url
  const launch = () => {
    try {
      if (!openPic365Window(startURL)) {
        handleServerError(
          new Error('Pic365 pop-up was blocked'),
          t('Pic365 pop-up was blocked')
        )
      } else {
        onClick?.()
      }
    } catch (error) {
      handleServerError(error, t('Pic365 pop-up was blocked'))
    }
  }
  if (presentation === 'sidebar') {
    return <Pic365SidebarLaunch onClick={launch} />
  }
  if (presentation === 'navigation') {
    return (
      <Button variant='ghost' size='sm' className={className} onClick={launch}>
        <Images className='size-4' aria-hidden='true' />
        {t('Pic365')}
      </Button>
    )
  }
  return (
    <Button variant='outline' size='sm' onClick={launch}>
      <Images className='size-4' />
      {t('Pic365')}
    </Button>
  )
}

export function Pic365Authorize(props: { state: string; challenge: string }) {
  const { t } = useTranslation()
  const [failed, setFailed] = useState(false)
  const request = useRef<Promise<string> | null>(null)
  useEffect(() => {
    let active = true
    if (!request.current) {
      request.current = api
        .post(
          '/api/user/self/pic365/authorize',
          {
            state: props.state,
            code_challenge: props.challenge,
          },
          { singleUseAuthorization: true }
        )
        .then((response) => {
          const payload = requireServerSuccess(response.data).data as {
            redirect_url: string
          }
          return payload.redirect_url
        })
    }
    request.current
      .then((url) => {
        if (active) window.location.replace(url)
      })
      .catch(() => {
        if (active) setFailed(true)
      })
    return () => {
      active = false
    }
  }, [props.state, props.challenge])
  if (failed) {
    return <ErrorState title={t('Pic365 authorization is unavailable')} />
  }
  return <LoadingState message={t('Opening Pic365')} />
}
