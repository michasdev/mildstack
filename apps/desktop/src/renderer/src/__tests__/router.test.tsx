import { describe, it, expect } from 'vitest'
import { createHashRouter } from 'react-router'

describe('router configuration', () => {
  it('creates a hash router without throwing', () => {
    expect(() =>
      createHashRouter([
        {
          path: '/',
          element: null,
          children: [
            { index: true, element: null },
            { path: '*', element: null },
          ],
        },
      ])
    ).not.toThrow()
  })

  it('router module exports a router object', async () => {
    const mod = await import('@renderer/router')
    expect(mod.router).toBeDefined()
    expect(typeof mod.router).toBe('object')
  })
})
