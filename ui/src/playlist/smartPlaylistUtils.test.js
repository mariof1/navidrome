import { parseCriteriaToForm } from './smartPlaylistUtils'

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
})

