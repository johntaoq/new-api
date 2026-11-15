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
import { describe, expect, test } from 'vitest'

import { channelSchema } from '../../types'
import {
  getChannelConfigurationSection,
  getChannelConfigurationState,
} from '../channel-configuration'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from '../channel-form'

describe('Copilot response metadata channel setting', () => {
  test.each([1, 14])(
    'provider %s loads the saved switch and retains other settings when turned off',
    (type) => {
      const channel = channelSchema.parse({
        id: 12,
        name: 'Metadata channel',
        key: '',
        type,
        status: 1,
        created_time: 1,
        test_time: 0,
        response_time: 0,
        balance_updated_time: 0,
        models: 'test-model',
        group: 'default',
        settings: JSON.stringify({
          strip_copilot_metadata: true,
          custom_option: { enabled: true },
        }),
      })
      const form = transformChannelToFormDefaults(channel)
      expect(form).toHaveProperty('strip_copilot_metadata', true)
      expect(
        getChannelConfigurationState(form, {}, true).sections.request
      ).toBe('configured')

      const changed = channelFormSchema.parse({
        ...form,
        strip_copilot_metadata: false,
      })
      const payload = transformFormDataToUpdatePayload(changed, channel.id)
      expect(JSON.parse(payload.settings ?? '{}')).toMatchObject({
        strip_copilot_metadata: false,
        custom_option: { enabled: true },
      })
      expect(payload).not.toHaveProperty('strip_copilot_metadata')
      expect(payload).not.toHaveProperty('key')
      expect(
        getChannelConfigurationState(changed, {}, true).sections.request
      ).toBe('idle')
    }
  )

  test.each([true, false])(
    'new channels save an enabled value of %s without adding a false default',
    (enabled) => {
      const form = channelFormSchema.parse({
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'Metadata channel',
        type: 1,
        key: 'test-key',
        models: 'test-model',
        settings: '{"custom_option":7}',
        strip_copilot_metadata: enabled,
      })
      const payload = transformFormDataToCreatePayload(form)
      const settings = JSON.parse(payload.channel.settings ?? '{}')
      expect(settings.custom_option).toBe(7)
      if (enabled) {
        expect(settings.strip_copilot_metadata).toBe(true)
      } else {
        expect(settings).not.toHaveProperty('strip_copilot_metadata')
      }
    }
  )

  test('a legacy channel without the setting stays off and saves without adding it', () => {
    const channel = channelSchema.parse({
      id: 11,
      name: 'Legacy channel',
      key: '',
      type: 1,
      status: 1,
      created_time: 1,
      test_time: 0,
      response_time: 0,
      balance_updated_time: 0,
      models: 'test-model',
      group: 'default',
      settings: '{"custom_option":7}',
    })
    const form = transformChannelToFormDefaults(channel)
    expect(form).toHaveProperty('strip_copilot_metadata', false)
    const payload = transformFormDataToUpdatePayload(form, channel.id)
    expect(JSON.parse(payload.settings ?? '{}')).not.toHaveProperty(
      'strip_copilot_metadata'
    )
  })

  test('the switch rejects string values instead of interpreting them as true', () => {
    const result = channelFormSchema.safeParse({
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'Metadata channel',
      type: 1,
      models: 'test-model',
      strip_copilot_metadata: 'true',
    })
    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ path: ['strip_copilot_metadata'] }),
        ])
      )
    }
  })

  test('the switch belongs to Request & Response for configuration and error navigation', () => {
    expect(getChannelConfigurationSection('strip_copilot_metadata')).toBe(
      'request'
    )
  })
})
