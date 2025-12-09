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

  it('extracts include and exclude strings as arrays', () => {
    const form = parseCriteriaToForm({
      all: [
        { contains: { artist: 'Artist 1' } },
        { notContains: { album: 'Album 1' } },
        { contains: { genre: 'Rock' } },
        { notContains: { genre: 'Metal' } },
      ],
    })

    expect(form.includeArtists).toEqual(['Artist 1'])
    expect(form.excludeAlbums).toEqual(['Album 1'])
    expect(form.includeGenres).toEqual(['Rock'])
    expect(form.excludeGenres).toEqual(['Metal'])
  })

  it('flattens nested conjunctions when extracting strings', () => {
    const form = parseCriteriaToForm({
      all: [
        { any: [{ contains: { artist: 'Artist 1' } }, { contains: { artist: 'Artist 2' } }] },
        { all: [{ contains: { genre: 'Rock' } }] },
      ],
    })

    expect(form.includeArtists).toEqual(['Artist 1', 'Artist 2'])
    expect(form.includeGenres).toEqual(['Rock'])
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

  it('builds include and exclude expressions', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      includeArtists: ['Artist 1', 'Artist 2'],
      excludeAlbums: ['Album 1'],
      includeGenres: ['Rock'],
      excludeGenres: ['Metal'],
    })

    expect(criteria.all).toEqual(
      expect.arrayContaining([
        {
          any: [
            { contains: { artist: 'Artist 1' } },
            { contains: { artist: 'Artist 2' } },
          ],
        },
        { notContains: { album: 'Album 1' } },
        { contains: { genre: 'Rock' } },
        { notContains: { genre: 'Metal' } },
      ])
    )
  })

  it('requires all included genres', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      includeGenres: ['Rock', 'Metal'],
    })

    expect(criteria.all).toEqual([
      { contains: { genre: 'Rock' } },
      { contains: { genre: 'Metal' } },
    ])
  })
})

