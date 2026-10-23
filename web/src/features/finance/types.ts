export type FinanceTab =
  | 'dashboard'
  | 'revenue'
  | 'channel'
  | 'customer'
  | 'audit'
export type FinanceUnit = 'usd' | 'cos'
export type StatementUnit = FinanceUnit | 'cny'
export type FinanceParams = Record<string, string | number>
export type FinanceRow = Record<string, string | number | null>
export type FinancePage = {
  page: number
  page_size: number
  total: number
  items: FinanceRow[]
  summary?: Record<string, number>
}
export type FinanceAmount = { usd: number; cos: number }
export type DashboardSummary = {
  kpis: {
    sales_income: FinanceAmount
    gift_granted: FinanceAmount
    channel_cost: FinanceAmount
    current_paid_balance: FinanceAmount
    current_gift_balance: FinanceAmount
    active_customer_count: number
    paid_customer_count: number
  }
  alerts: FinanceRow[]
}
export type RevenueSummary = {
  paid_recharge_total: FinanceAmount
  gift_granted_total: FinanceAmount
  unissued_paid_voucher_total: FinanceAmount
  unredeemed_gift_voucher_total: FinanceAmount
}
export type ChannelSummary = {
  kpis: { total_cost_usd: number; total_tokens: number }
  charts: Record<
    'by_channel_usd' | 'by_channel_tokens' | 'by_model_usd' | 'by_model_tokens',
    { name: string; value: number }[]
  >
}
export type FinanceColumn = {
  key: string
  label: string
  kind?: 'money' | 'number' | 'date' | 'label'
  cosKey?: string
  cnyKey?: string
}

export type FinanceBillingMonth = {
  bill_month: string
  cny_rate: number
  cos_rate: number
  closes_at: number
  finalized_at: number
  last_error: string
}

export type FinanceMonthlyPolicy = {
  current_month: string
  next_month: string
  months: FinanceBillingMonth[]
}
