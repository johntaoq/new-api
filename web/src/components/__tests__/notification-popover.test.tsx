import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterAll, beforeAll, describe, expect, test } from 'vitest'

import { NotificationPopover } from '@/components/notification-popover'

function NotificationExample(props: { loading?: boolean; content?: boolean }) {
  const [open, setOpen] = useState(true)
  const [tab, setTab] = useState<'notice' | 'announcements'>('notice')
  return (
    <NotificationPopover
      open={open}
      onOpenChange={setOpen}
      unreadCount={0}
      activeTab={tab}
      onTabChange={setTab}
      notice={props.content ? 'Account notice\n\n'.repeat(40) : ''}
      announcements={
        props.content
          ? [{ id: 1, content: 'Platform update\n\n'.repeat(40) }]
          : []
      }
      loading={props.loading ?? false}
    />
  )
}

describe('notification viewport layout', () => {
  const originalAnimations = Object.getOwnPropertyDescriptor(
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
    if (originalAnimations) {
      Object.defineProperty(
        HTMLElement.prototype,
        'getAnimations',
        originalAnimations
      )
    } else {
      Reflect.deleteProperty(HTMLElement.prototype, 'getAnimations')
    }
  })

  test.each([false, true])(
    'empty or loading notifications keep the original width and fill the viewport (loading=%s)',
    async (loading) => {
      render(<NotificationExample loading={loading} />)
      const dialog = await screen.findByRole('dialog', {
        name: 'System Announcements',
      })
      expect(dialog).toHaveClass(
        'w-[min(26rem,calc(100vw-1rem))]',
        'h-dvh',
        'overflow-hidden'
      )
      expect(within(dialog).getByRole('tablist')).toHaveClass('shrink-0')
      const close = within(dialog).getByRole('button', { name: 'Close' })
      expect(close.parentElement).toHaveClass('shrink-0')
      await userEvent.click(close)
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    }
  )

  test('long notices and timeline entries scroll inside the remaining height and still close with Escape', async () => {
    render(<NotificationExample content />)
    const dialog = await screen.findByRole('dialog', {
      name: 'System Announcements',
    })
    const noticePanel = within(dialog).getByRole('tabpanel')
    expect(noticePanel).toHaveClass('min-h-0')
    expect(noticePanel.querySelector('[data-slot=scroll-area]')).toHaveClass(
      'h-full'
    )
    await userEvent.click(within(dialog).getByRole('tab', { name: 'Timeline' }))
    const timelinePanel = within(dialog).getByRole('tabpanel')
    expect(timelinePanel).toHaveClass('min-h-0')
    expect(timelinePanel.querySelector('[data-slot=scroll-area]')).toHaveClass(
      'h-full'
    )
    expect(within(dialog).getByRole('button', { name: 'Close' })).toBeVisible()
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
