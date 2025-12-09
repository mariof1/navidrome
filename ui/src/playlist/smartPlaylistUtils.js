const extractRange = (rules = [], field) => {
  let min
  let max
  rules.forEach((rule) => {
    if (rule.inTheRange && rule.inTheRange[field]) {
      const [rMin, rMax] = rule.inTheRange[field]
      min = rMin
      max = rMax
      return
    }
    if (rule.gt && rule.gt[field] !== undefined) {
      min = rule.gt[field]
    }
    if (rule.lt && rule.lt[field] !== undefined) {
      max = rule.lt[field]
    }
  })
  return { min, max }
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
  const playCountRange = extractRange(rules, 'playcount')

  return {
    smart: true,
    minDuration: durationRange.min,
    maxDuration: durationRange.max,
    minPlayCount: playCountRange.min,
    maxPlayCount: playCountRange.max,
    artist: extractString(rules, 'artist'),
    album: extractString(rules, 'album'),
    genre: extractString(rules, 'genre'),
    sort: criteria.sort?.toLowerCase(),
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

  const expressions = [
    ...buildRangeExpressions('duration', formData.minDuration, formData.maxDuration),
    ...buildRangeExpressions('playcount', formData.minPlayCount, formData.maxPlayCount),
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

  if (formData.sort) {
    criteria.sort = formData.sort
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

