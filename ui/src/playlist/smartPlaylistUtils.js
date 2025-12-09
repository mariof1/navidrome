const extractRange = (rules = [], fields) => {
  let min
  let max
  const fieldsArray = Array.isArray(fields) ? fields : [fields]
  rules.forEach((rule) => {
    if (rule.inTheRange) {
      for (const field of fieldsArray) {
        if (rule.inTheRange[field]) {
          const [rMin, rMax] = rule.inTheRange[field]
          min = rMin
          max = rMax
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

const extractString = (rules = [], field) => {
  const match = rules.find((rule) => rule.contains && rule.contains[field])
  return match?.contains[field]
}

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
    artist: extractString(rules, 'artist'),
    album: extractString(rules, 'album'),
    genre: extractString(rules, 'genre'),
    sort: sortData.sort,
    order: criteria.order?.toLowerCase(),
    trackLimit: criteria.limit,
  }
}

const buildRangeExpressions = (field, min, max) => {
  if (min === undefined && max === undefined) {
    return []
  }
  if (min !== undefined && min !== null && min !== '' && max !== undefined && max !== null && max !== '') {
    return [{ inTheRange: { [field]: [Number(min), Number(max)] } }]
  }
  if (min !== undefined && min !== null && min !== '') {
    return [{ gt: { [field]: Number(min) } }]
  }
  if (max !== undefined && max !== null && max !== '') {
    return [{ lt: { [field]: Number(max) } }]
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

  if (formData.artist) {
    expressions.push({ contains: { artist: formData.artist } })
  }
  if (formData.album) {
    expressions.push({ contains: { album: formData.album } })
  }
  if (formData.genre) {
    expressions.push({ contains: { genre: formData.genre } })
  }

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
    artist,
    album,
    genre,
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
    artist,
    album,
    genre,
    sort,
    order,
    trackLimit,
    core: rest,
  }
}

export const buildPlaylistPayload = (formData) => {
  const { core } = stripSmartFormFields(formData)
  const rules = buildSmartCriteria(formData)

  return {
    ...core,
    rules,
  }
}

