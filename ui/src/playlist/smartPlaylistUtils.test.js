import { buildSmartCriteria, parseCriteriaToForm } from './smartPlaylistUtils'

describe('parseCriteriaToForm', () => {
  it('normalizes sort and order values to lower case', () => {
    const form = parseCriteriaToForm({
      all: [{ gt: { duration: 0 } }],
      sort: 'PlayCount',
      order: 'DESC',
      limit: 25,
    })

    expect(form.sort).toBe('playcount')
    expect(form.order).toBe('desc')
    expect(form.trackLimit).toBe(25)
  })

  it('extracts playcount from all users ranges and sorts', () => {
    const form = parseCriteriaToForm({
      all: [{ inTheRange: { playcountallusers: [1, 10] } }],
      sort: 'playCountAllUsers',
    })

    expect(form.includeAllUsersPlayCount).toBe(true)
    expect(form.minPlayCount).toBe(1)
    expect(form.maxPlayCount).toBe(10)
    expect(form.sort).toBe('playcount')
  })
})

describe('buildSmartCriteria', () => {
  it('uses aggregated play count when requested', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      includeAllUsersPlayCount: true,
      minPlayCount: 5,
      sort: 'playcount',
    })

    expect(criteria.all[0]).toEqual({ gt: { playcountallusers: 5 } })
    expect(criteria.sort).toBe('playcountallusers')
  })
})

