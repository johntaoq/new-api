import { describe, expect, it } from 'vitest'

import { getStaticHomePath } from '../lib/static-home'

describe('getStaticHomePath', () => {
  it('returns the server-served homepage for the 99Off API host', () => {
    expect(getStaticHomePath('api.99off.net')).toBe('/')
  })

  it('keeps the NewAPI homepage for other hosts', () => {
    expect(getStaticHomePath('www.99off.net')).toBeNull()
    expect(getStaticHomePath('api.unikeyx.com')).toBeNull()
    expect(getStaticHomePath('localhost')).toBeNull()
  })
})
