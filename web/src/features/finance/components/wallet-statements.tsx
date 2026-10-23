import { useMutation } from '@tanstack/react-query'
import { Download, ReceiptText } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { exportStatement } from '../api'
import { FinanceCustomers } from './customers'
import { FinanceRemoteTable } from './shared'

export function WalletMonthlyStatements() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const download = useMutation({
    mutationFn: () => exportStatement({}, true, true),
  })
  return (
    <div>
      <Button variant='outline' onClick={() => setOpen(true)}>
        <ReceiptText />
        {t('Monthly statements')}
      </Button>
      <Dialog
        open={open}
        onOpenChange={setOpen}
        title={t('Monthly statements')}
        contentClassName='sm:max-w-6xl'
      >
        {open && (
          <Tabs defaultValue='fixed'>
            <TabsList>
              <TabsTrigger value='fixed'>
                {t('Finalized monthly statements')}
              </TabsTrigger>
              <TabsTrigger value='current'>
                {t('Current month details')}
              </TabsTrigger>
            </TabsList>
            <TabsContent value='fixed'>
              <FinanceCustomers self />
            </TabsContent>
            <TabsContent value='current' className='space-y-4'>
              <div className='flex flex-wrap items-center justify-between gap-3'>
                <span className='text-muted-foreground text-sm'>
                  {t('Not finalized')}
                </span>
                <Button
                  variant='outline'
                  disabled={download.isPending}
                  onClick={() => download.mutate()}
                >
                  <Download />
                  {t('Download CNY details')}
                </Button>
              </div>
              <FinanceRemoteTable
                title='Current month details'
                path='self/current'
                params={{}}
                unit='cny'
                columns={[
                  { key: 'entry_type', label: 'Type', kind: 'label' },
                  {
                    key: 'amount_cny',
                    cnyKey: 'amount_cny',
                    label: 'Balance change (CNY)',
                    kind: 'money',
                  },
                  { key: 'occurred_at', label: 'Time', kind: 'date' },
                  { key: 'token_display', label: 'Token' },
                  { key: 'model_name', label: 'Model' },
                  { key: 'request_id', label: 'Request ID' },
                ]}
              />
            </TabsContent>
          </Tabs>
        )}
      </Dialog>
    </div>
  )
}
