import { useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { hasPermission } from '@/lib/admin-permissions'
import dayjs from '@/lib/dayjs'
import { useAuthStore } from '@/stores/auth-store'

import { FinanceAudit } from './components/audit'
import { FinanceChannels } from './components/channels'
import { FinanceCustomers } from './components/customers'
import { FinanceOverview } from './components/overview'
import { FinanceRevenue } from './components/revenue'
import { FinanceSegments } from './components/shared'
import type { FinanceTab, FinanceUnit } from './types'

export function FinanceCenter() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const client = useQueryClient()
  const canRead = hasPermission(user, 'finance', 'read')
  const canAudit = hasPermission(user, 'finance', 'audit')
  const [tab, setTab] = useState<FinanceTab>(canRead ? 'dashboard' : 'audit')
  const [periodType, setPeriodType] = useState('month')
  const [period, setPeriod] = useState(dayjs().format('YYYY-MM'))
  const [periodDraft, setPeriodDraft] = useState(period)
  const [unit, setUnit] = useState<FinanceUnit>('usd')
  const tabs: { value: FinanceTab; label: string }[] = []
  if (canRead) {
    tabs.push(
      { value: 'dashboard', label: 'Operating dashboard' },
      { value: 'revenue', label: 'Revenue and gifts' },
      { value: 'channel', label: 'Channel costs' },
      { value: 'customer', label: 'Customer statements' }
    )
  }
  if (canAudit) tabs.push({ value: 'audit', label: 'Financial audit' })
  const activeTab = tabs.some((item) => item.value === tab)
    ? tab
    : tabs[0]?.value
  const params = { period_type: periodType, period }
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Finance Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          size='icon'
          variant='outline'
          aria-label={t('Refresh')}
          title={t('Refresh')}
          onClick={() =>
            void client.invalidateQueries({ queryKey: ['finance'] })
          }
        >
          <RefreshCw />
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setTab(value as FinanceTab)}
          className='min-w-0 gap-5'
        >
          <div className='max-w-full overflow-x-auto pb-1'>
            <TabsList variant='line' className='h-10'>
              {tabs.map((item) => (
                <TabsTrigger
                  className='px-3'
                  key={item.value}
                  value={item.value}
                >
                  {t(item.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
          <div className='flex min-w-0 flex-wrap items-center justify-between gap-3'>
            {activeTab !== 'customer' && (
              <div className='flex min-w-0 flex-wrap items-center gap-2'>
                <FinanceSegments
                  label='Period type'
                  value={periodType}
                  onChange={(value) => {
                    setPeriodType(value)
                    const next = dayjs().format(
                      value === 'year' ? 'YYYY' : 'YYYY-MM'
                    )
                    setPeriod(next)
                    setPeriodDraft(next)
                  }}
                  items={[
                    { value: 'month', label: 'Monthly' },
                    { value: 'year', label: 'Yearly' },
                  ]}
                />
                <Input
                  type={periodType === 'month' ? 'month' : 'number'}
                  aria-label={t('Reporting period')}
                  min={periodType === 'year' ? 2000 : undefined}
                  max={periodType === 'year' ? 9999 : undefined}
                  value={periodDraft}
                  className='w-44'
                  onBlur={() => setPeriodDraft(period)}
                  onChange={(event) => {
                    const value = event.target.value
                    setPeriodDraft(value)
                    if (
                      (periodType === 'year' && /^\d{4}$/.test(value)) ||
                      (periodType === 'month' && /^\d{4}-\d{2}$/.test(value))
                    ) {
                      setPeriod(value)
                    }
                  }}
                />
              </div>
            )}
            {activeTab !== 'audit' &&
              activeTab !== 'channel' &&
              activeTab !== 'customer' && (
                <FinanceSegments
                  label='Display currency'
                  value={unit}
                  onChange={(value) => setUnit(value as FinanceUnit)}
                  items={[
                    { value: 'usd', label: 'USD' },
                    { value: 'cos', label: 'COS' },
                  ]}
                />
              )}
          </div>
          {canRead && (
            <>
              <TabsContent value='dashboard'>
                <FinanceOverview
                  key={`${periodType}-${period}`}
                  params={params}
                  unit={unit}
                />
              </TabsContent>
              <TabsContent value='revenue'>
                <FinanceRevenue
                  key={`${periodType}-${period}`}
                  params={params}
                  unit={unit}
                />
              </TabsContent>
              <TabsContent value='channel'>
                <FinanceChannels
                  key={`${periodType}-${period}`}
                  params={params}
                />
              </TabsContent>
              <TabsContent value='customer'>
                <FinanceCustomers />
              </TabsContent>
            </>
          )}
          {canAudit && (
            <TabsContent value='audit'>
              <FinanceAudit key={`${periodType}-${period}`} params={params} />
            </TabsContent>
          )}
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
