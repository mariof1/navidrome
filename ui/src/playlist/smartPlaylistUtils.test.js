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
    expect(form.includeArtistsMatchMode).toBe('any')
    expect(form.excludeAlbums).toEqual(['Album 1'])
    expect(form.includeGenres).toEqual(['Rock'])
    expect(form.includeGenresMatchMode).toBe('any')
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
    expect(form.includeArtistsMatchMode).toBe('any')
    expect(form.includeGenres).toEqual(['Rock'])
    expect(form.includeGenresMatchMode).toBe('any')
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

    expect(criteria.all[0]).toEqual({ inTheRange: { playcountallusers: [5, Number.MAX_SAFE_INTEGER] } })
    expect(criteria.sort).toBe('playcountallusers')
  })

  it('creates inclusive ranges for playcount bounds', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      maxPlayCount: 0,
    })

    expect(criteria.all[0]).toEqual({ inTheRange: { playcount: [Number.MIN_SAFE_INTEGER, 0] } })
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

  it('builds match-all expressions when requested', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      includeGenres: ['Rock', 'Metal'],
      includeGenresMatchMode: 'all',
      includeArtists: ['Artist 1', 'Artist 2'],
      includeArtistsMatchMode: 'all',
    })

    expect(criteria.all).toEqual(
      expect.arrayContaining([
        { contains: { genre: ['Rock', 'Metal'] } },
        { contains: { artist: ['Artist 1', 'Artist 2'] } },
      ])
    )
    expect(criteria.genresMatchMode).toBe('all')
    expect(criteria.artistsMatchMode).toBe('all')
  })

  it('preserves match-any and match-all modes for different fields', () => {
    const criteria = buildSmartCriteria({
      smart: true,
      includeGenres: ['Rock', 'Metal'],
      includeGenresMatchMode: 'any',
      includeAlbums: ['Album 1', 'Album 2'],
      includeAlbumsMatchMode: 'all',
    })

    expect(criteria.all).toEqual(
      expect.arrayContaining([
        {
          any: [
            { contains: { genre: 'Rock' } },
            { contains: { genre: 'Metal' } },
          ],
        },
        { contains: { album: ['Album 1', 'Album 2'] } },
      ])
    )
    expect(criteria.genresMatchMode).toBe('any')
    expect(criteria.albumsMatchMode).toBe('all')
  })

  describe('play count ranges', () => {
    it('omits playcount filters when both bounds are empty', () => {
      const criteria = buildSmartCriteria({
        smart: true,
      })

      expect(criteria.all).not.toEqual(
        expect.arrayContaining([expect.objectContaining({ inTheRange: expect.anything() })])
      )
    })

    it('filters by minimum only when max is unset', () => {
      const criteria = buildSmartCriteria({
        smart: true,
        minPlayCount: 5,
      })

      expect(criteria.all).toEqual([
        { inTheRange: { playcount: [5, Number.MAX_SAFE_INTEGER] } },
      ])
    })

    it('filters by maximum only when min is unset', () => {
      const criteria = buildSmartCriteria({
        smart: true,
        maxPlayCount: 3,
      })

      expect(criteria.all).toEqual([
        { inTheRange: { playcount: [Number.MIN_SAFE_INTEGER, 3] } },
      ])
    })

    it('filters between inclusive bounds when both are set', () => {
      const criteria = buildSmartCriteria({
        smart: true,
        minPlayCount: 1,
        maxPlayCount: 3,
      })

      expect(criteria.all).toEqual([{ inTheRange: { playcount: [1, 3] } }])
    })

    it('matches never-played tracks when both bounds are zero', () => {
      const criteria = buildSmartCriteria({
        smart: true,
        minPlayCount: 0,
        maxPlayCount: 0,
      })

      expect(criteria.all).toEqual([{ inTheRange: { playcount: [0, 0] } }])
    })
  })
})

