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
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { PerformanceHealthPanel } from '../performance-health-panel'

let client: QueryClient
beforeEach(() => {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
})
afterEach(() => client.clear())

const summary = { success_rate: 96.81, avg_latency_ms: 11500, avg_tps: 114.3 }

describe('performance health panel model visibility', () => {
  it('hides image 1.5 and fills six model slots without changing the summary or cached models', () => {
    const modelNames = [
      'gpt-image-1.5',
      'gpt-chat-latest',
      'claude-opus-5-5',
      'gpt-6-luna',
      'gpt-6.1-sol',
      'gpt-6-astra',
      'gpt-5.6-sol',
    ]
    const models = modelNames.map((model_name) => ({
      model_name,
      success_rate: model_name === 'gpt-image-1.5' ? 0 : 100,
    }))
    client.setQueryData(['perf-metrics-summary', 24], {
      success: true,
      data: { summary, models },
    })
    render(
      <QueryClientProvider client={client}>
        <PerformanceHealthPanel />
      </QueryClientProvider>
    )

    expect(screen.queryByText('gpt-image-1.5')).not.toBeInTheDocument()
    for (const name of modelNames.slice(1)) {
      expect(screen.getByText(name)).toBeVisible()
    }
    expect(screen.getByText('96.81%')).toBeVisible()
    expect(models.map((model) => model.model_name)).toEqual(modelNames)
  })

  it('omits the model list when only image 1.5 has samples', () => {
    client.setQueryData(['perf-metrics-summary', 24], {
      success: true,
      data: {
        summary,
        models: [{ model_name: 'gpt-image-1.5', success_rate: 0 }],
      },
    })
    render(
      <QueryClientProvider client={client}>
        <PerformanceHealthPanel />
      </QueryClientProvider>
    )

    expect(screen.queryByText('gpt-image-1.5')).not.toBeInTheDocument()
    expect(screen.queryByText('Top models by traffic')).not.toBeInTheDocument()
    expect(screen.getByText('96.81%')).toBeVisible()
  })
})
