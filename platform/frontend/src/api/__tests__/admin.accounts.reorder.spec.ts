import { beforeEach, describe, expect, it, vi } from 'vitest'

const { put } = vi.hoisted(() => ({ put: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { put } }))
import accountsAPI, { reorder } from '@/api/admin/accounts'

describe('admin account display order API', () => {
  beforeEach(() => { put.mockReset() })

  it('PUTs the complete ordered ID list without changing account priorities', async () => {
    const response = { success: true }
    put.mockResolvedValueOnce({ data: response })
    await expect(reorder([31, 7, 24])).resolves.toEqual(response)
    expect(put).toHaveBeenCalledTimes(1)
    expect(put).toHaveBeenCalledWith('/admin/accounts/reorder', { ids: [31, 7, 24] })
    expect(accountsAPI.reorder).toBe(reorder)
  })

  it('propagates server rejection for the dialog to report and retry', async () => {
    const failure = { status: 409, message: 'Account list changed' }
    put.mockRejectedValueOnce(failure)
    await expect(accountsAPI.reorder([7, 31])).rejects.toBe(failure)
    expect(put).toHaveBeenCalledTimes(1)
  })
})
