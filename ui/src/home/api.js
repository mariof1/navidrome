import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'

export const getHomeRecommendations = async ({ limit, seed } = {}) => {
  const params = new URLSearchParams()
  if (limit) params.set('limit', String(limit))
  if (seed) params.set('seed', String(seed))

  const query = params.toString()
  const url = query ? `${REST_URL}/recommendations/home?${query}` : `${REST_URL}/recommendations/home`

  const { json } = await httpClient(url)
  return json
}
