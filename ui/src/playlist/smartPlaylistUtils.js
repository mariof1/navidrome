const RANGE_MIN = Number.MIN_SAFE_INTEGER
const RANGE_MAX = Number.MAX_SAFE_INTEGER

const flattenRules = (rules = []) =>
  rules.flatMap((rule) => {
    if (rule.all) {
      return flattenRules(rule.all)
    }
    if (rule.any) {
      return flattenRules(rule.any)
    }
    return [rule]
  })

const normalizeRangeValue = (value, bound) =>
  value === bound ? undefined : value

const extractRange = (rules = [], fields) => {
  let min
  let max
  const fieldsArray = Array.isArray(fields) ? fields : [fields]
  rules.forEach((rule) => {
    if (rule.inTheRange) {
      for (const field of fieldsArray) {
        if (rule.inTheRange[field]) {
          const [rMin, rMax] = rule.inTheRange[field]
          min = normalizeRangeValue(rMin, RANGE_MIN)
          max = normalizeRangeValue(rMax, RANGE_MAX)
          return
        }
      }
    }
    fieldsArray.forEach((field) => {
      if (rule.gt && rule.gt[field] !== undefined) {
        min = rule.gt[field]
      }
      if (rule.lt && rule.lt[field] !== undefined) {
        max = rule.lt[field]
      }
    })
  })
  const field = fieldsArray.find(
    (candidate) =>
      rules.some(
        (rule) =>
          (rule.inTheRange && rule.inTheRange[candidate]) ||
          (rule.gt && rule.gt[candidate] !== undefined) ||
          (rule.lt && rule.lt[candidate] !== undefined)
      )
  )
  return { min, max, field }
}

const extractSort = (sort) => {
  const normalizedSort = sort?.toLowerCase()
  if (normalizedSort === 'playcountallusers') {
    return { sort: 'playcount', useAllUsers: true }
  }
  if (normalizedSort === 'playcount') {
    return { sort: normalizedSort, useAllUsers: false }
  }
  return { sort: normalizedSort, useAllUsers: false }
}

const extractStrings = (rules = [], field, operator = 'contains') =>
  flattenRules(rules)
    .filter((rule) => rule[operator] && rule[operator][field])
    .map((rule) => rule[operator][field])

export const parseCriteriaToForm = (criteria) => {
  if (!criteria?.expression && !criteria?.Expression && !criteria?.all && !criteria?.any) {
    return {}
  }

  const expr = criteria.all || criteria.any || criteria.Expression || criteria.expression
  const rules = Array.isArray(expr) ? expr : []

  const durationRange = extractRange(rules, 'duration')
  const playCountRange = extractRange(rules, ['playcountallusers', 'playcount'])
  const sortData = extractSort(criteria.sort)
  const includeAllUsersPlayCount =
    playCountRange.field === 'playcountallusers' || sortData.useAllUsers

  return {
    smart: true,
    minDuration: durationRange.min,
    maxDuration: durationRange.max,
    minPlayCount: playCountRange.min,
    maxPlayCount: playCountRange.max,
    includeAllUsersPlayCount,
    includeArtists: extractStrings(rules, 'artist'),
    excludeArtists: extractStrings(rules, 'artist', 'notContains'),
    includeAlbums: extractStrings(rules, 'album'),
    excludeAlbums: extractStrings(rules, 'album', 'notContains'),
    includeGenres: extractStrings(rules, 'genre'),
    excludeGenres: extractStrings(rules, 'genre', 'notContains'),
    sort: sortData.sort,
    order: criteria.order?.toLowerCase(),
    trackLimit: criteria.limit,
  }
}

const normalizeStrings = (value) => {
  if (value === undefined || value === null) {
    return []
  }

  if (Array.isArray(value)) {
    return value
      .filter((entry) => entry !== undefined && entry !== null)
      .map((entry) => `${entry}`.trim())
      .filter(Boolean)
  }

  const normalized = `${value}`.trim()
  return normalized ? [normalized] : []
}

const addOrStringExpressions = (expressions, values, operator, field) => {
  const normalized = normalizeStrings(values)
  if (normalized.length === 0) {
    return
  }
  if (normalized.length === 1) {
    expressions.push({ [operator]: { [field]: normalized[0] } })
    return
  }
  expressions.push({
    any: normalized.map((value) => ({ [operator]: { [field]: value } })),
  })
}

const addStringExpressions = (expressions, values, operator, field) => {
  normalizeStrings(values).forEach((value) => {
    expressions.push({ [operator]: { [field]: value } })
  })
}

const buildRangeExpressions = (field, min, max) => {
  if (min === undefined && max === undefined) {
    return []
  }
  const hasMin = min !== undefined && min !== null && min !== ''
  const hasMax = max !== undefined && max !== null && max !== ''

  if (hasMin || hasMax) {
    return [
      {
        inTheRange: {
          [field]: [
            hasMin ? Number(min) : RANGE_MIN,
            hasMax ? Number(max) : RANGE_MAX,
          ],
        },
      },
    ]
  }

  return []
}

export const buildSmartCriteria = (formData) => {
  if (!formData.smart) {
    return null
  }

  const playCountField = formData.includeAllUsersPlayCount
    ? 'playcountallusers'
    : 'playcount'
  const sortField =
    formData.sort === 'playcount' && formData.includeAllUsersPlayCount
      ? 'playcountallusers'
      : formData.sort

  const expressions = [
    ...buildRangeExpressions('duration', formData.minDuration, formData.maxDuration),
    ...buildRangeExpressions(playCountField, formData.minPlayCount, formData.maxPlayCount),
  ]

  addOrStringExpressions(expressions, formData.includeArtists, 'contains', 'artist')
  addStringExpressions(expressions, formData.excludeArtists, 'notContains', 'artist')
  addOrStringExpressions(expressions, formData.includeAlbums, 'contains', 'album')
  addStringExpressions(expressions, formData.excludeAlbums, 'notContains', 'album')
  addStringExpressions(expressions, formData.includeGenres, 'contains', 'genre')
  addStringExpressions(expressions, formData.excludeGenres, 'notContains', 'genre')

  if (expressions.length === 0) {
    expressions.push({ gt: { duration: 0 } })
  }

  const criteria = {
    all: expressions,
  }

  if (sortField) {
    criteria.sort = sortField
  }
  if (formData.order) {
    criteria.order = formData.order
  }
  if (formData.trackLimit) {
    criteria.limit = Number(formData.trackLimit)
  }

  return criteria
}

export const stripSmartFormFields = (data) => {
  const {
    smart,
    minDuration,
    maxDuration,
    minPlayCount,
    maxPlayCount,
    includeAllUsersPlayCount,
    includeArtists,
    excludeArtists,
    includeAlbums,
    excludeAlbums,
    includeGenres,
    excludeGenres,
    sort,
    order,
    trackLimit,
    ...rest
  } = data

  return {
    smart,
    minDuration,
    maxDuration,
    minPlayCount,
    maxPlayCount,
    includeAllUsersPlayCount,
    includeArtists,
    excludeArtists,
    includeAlbums,
    excludeAlbums,
    includeGenres,
    excludeGenres,
    sort,
    order,
    trackLimit,
    core: rest,
  }
}

export const buildPlaylistPayload = (formData) => {
  const { core } = stripSmartFormFields(formData)
  const rules = buildSmartCriteria(formData)
  const sync = formData.smart ? false : core.sync

  return {
    ...core,
    ...(sync !== undefined ? { sync } : {}),
    rules,
  }
}

