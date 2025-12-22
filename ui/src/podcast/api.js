import httpClient from '../dataProvider/httpClient'
import { REST_URL } from '../consts'

const normalizeResponse = (json) => json?.data ?? json

export const listPodcasts = () =>
  httpClient(`${REST_URL}/podcast`).then(({ json }) => normalizeResponse(json))

export const createPodcastChannel = (data) =>
  httpClient(`${REST_URL}/podcast`, {
    method: 'POST',
    body: JSON.stringify(data),
  }).then(({ json }) => normalizeResponse(json))

export const getPodcastChannel = (id) =>
  httpClient(`${REST_URL}/podcast/${id}`).then(({ json }) => normalizeResponse(json))

export const listPodcastEpisodes = (id) =>
  httpClient(`${REST_URL}/podcast/${id}/episodes`).then(({ json }) =>
    normalizeResponse(json),
  )

export const updatePodcastChannel = (id, data) =>
  httpClient(`${REST_URL}/podcast/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  }).then(({ json }) => normalizeResponse(json))

export const deletePodcastChannel = (id) =>
  httpClient(`${REST_URL}/podcast/${id}`, { method: 'DELETE' }).then(({ json }) =>
    normalizeResponse(json),
  )

export const setEpisodeWatched = (channelId, episodeId, watched) =>
  httpClient(`${REST_URL}/podcast/${channelId}/episodes/${episodeId}/watched`, {
    method: 'PUT',
    body: JSON.stringify({ watched }),
  })

export const setEpisodeProgress = (channelId, episodeId, position, duration) =>
  httpClient(`${REST_URL}/podcast/${channelId}/episodes/${episodeId}/progress`, {
    method: 'PUT',
    body: JSON.stringify({ position, duration }),
  })

export const getEpisodeProgress = (channelId, episodeId) =>
  httpClient(`${REST_URL}/podcast/${channelId}/episodes/${episodeId}/progress`).then(
    ({ json }) => normalizeResponse(json),
  )

export const listContinueListening = (limit = 20) =>
  httpClient(`${REST_URL}/podcast/continue?limit=${limit}`).then(({ json }) =>
    normalizeResponse(json),
  )

export const searchApplePodcasts = (term, limit = 25) =>
  httpClient(
    `${REST_URL}/podcast/search?term=${encodeURIComponent(term)}&limit=${limit}`,
  ).then(({ json }) => normalizeResponse(json))
