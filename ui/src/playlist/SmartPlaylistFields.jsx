import React from 'react'
import { Grid, Typography } from '@material-ui/core'
import { BooleanInput, FormDataConsumer, NumberInput, SelectInput, useTranslate } from 'react-admin'
import SmartCriteriaAutocompleteArrayInput from './SmartCriteriaAutocompleteArrayInput'

const orderChoices = [
  { id: 'asc', name: 'resources.playlist.smart.order.asc' },
  { id: 'desc', name: 'resources.playlist.smart.order.desc' },
]

const sortChoices = [
  { id: 'random', name: 'resources.playlist.smart.sort.random' },
  { id: 'title', name: 'resources.playlist.smart.sort.title' },
  { id: 'album', name: 'resources.playlist.smart.sort.album' },
  { id: 'artist', name: 'resources.playlist.smart.sort.artist' },
  { id: 'duration', name: 'resources.playlist.smart.sort.duration' },
  { id: 'playcount', name: 'resources.playlist.smart.sort.playcount' },
  { id: 'lastplayed', name: 'resources.playlist.smart.sort.lastPlayed' },
  { id: 'dateadded', name: 'resources.playlist.smart.sort.dateAdded' },
]

const matchModeChoices = (translate) => [
  { id: 'any', name: translate('resources.playlist.smart.match.any') },
  { id: 'all', name: translate('resources.playlist.smart.match.all') },
]

const SmartPlaylistFields = () => {
  const translate = useTranslate()

  return (
    <>
      <BooleanInput source="smart" label="resources.playlist.fields.smart" />
      <FormDataConsumer>
        {({ formData }) =>
          formData.smart && (
            <Grid container spacing={3}>
              <Grid item xs={12}>
                <Typography variant="subtitle2" color="textSecondary">
                  {translate('resources.playlist.smart.sections.criteria')}
                </Typography>
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="minDuration"
                  label={translate('resources.playlist.smart.fields.minDuration')}
                  helperText={translate('resources.playlist.smart.help.duration')}
                  fullWidth
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="maxDuration"
                  label={translate('resources.playlist.smart.fields.maxDuration')}
                  helperText={translate('resources.playlist.smart.help.duration')}
                  fullWidth
                />
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="minPlayCount"
                  label={translate('resources.playlist.smart.fields.minPlayCount')}
                  parse={(value) => (value === '' || value === null ? undefined : value)}
                  fullWidth
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="maxPlayCount"
                  label={translate('resources.playlist.smart.fields.maxPlayCount')}
                  parse={(value) => (value === '' || value === null ? undefined : value)}
                  fullWidth
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <BooleanInput
                  source="includeAllUsersPlayCount"
                  label={translate('resources.playlist.smart.fields.playCountAllUsers')}
                />
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <Grid container direction="column" spacing={1}>
                  <Grid item>
                    <SmartCriteriaAutocompleteArrayInput
                      reference="artist"
                      source="includeArtists"
                      label="resources.playlist.smart.fields.includeArtist"
                    />
                  </Grid>
                  <Grid item>
                    <SelectInput
                      source="includeArtistsMatchMode"
                      label="resources.playlist.smart.match.label"
                      choices={matchModeChoices(translate)}
                      defaultValue="any"
                      fullWidth
                    />
                  </Grid>
                </Grid>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <Grid container direction="column" spacing={1}>
                  <Grid item>
                    <SmartCriteriaAutocompleteArrayInput
                      reference="album"
                      source="includeAlbums"
                      label="resources.playlist.smart.fields.includeAlbum"
                    />
                  </Grid>
                  <Grid item>
                    <SelectInput
                      source="includeAlbumsMatchMode"
                      label="resources.playlist.smart.match.label"
                      choices={matchModeChoices(translate)}
                      defaultValue="any"
                      fullWidth
                    />
                  </Grid>
                </Grid>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <Grid container direction="column" spacing={1}>
                  <Grid item>
                    <SmartCriteriaAutocompleteArrayInput
                      reference="genre"
                      source="includeGenres"
                      label="resources.playlist.smart.fields.includeGenre"
                    />
                  </Grid>
                  <Grid item>
                    <SelectInput
                      source="includeGenresMatchMode"
                      label="resources.playlist.smart.match.label"
                      choices={matchModeChoices(translate)}
                      defaultValue="any"
                      fullWidth
                    />
                  </Grid>
                </Grid>
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <SmartCriteriaAutocompleteArrayInput
                  reference="artist"
                  source="excludeArtists"
                  label="resources.playlist.smart.fields.excludeArtist"
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <SmartCriteriaAutocompleteArrayInput
                  reference="album"
                  source="excludeAlbums"
                  label="resources.playlist.smart.fields.excludeAlbum"
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <SmartCriteriaAutocompleteArrayInput
                  reference="genre"
                  source="excludeGenres"
                  label="resources.playlist.smart.fields.excludeGenre"
                />
              </Grid>

              <Grid item xs={12}>
                <Typography variant="subtitle2" color="textSecondary">
                  {translate('resources.playlist.smart.sections.results')}
                </Typography>
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="trackLimit"
                  label={translate('resources.playlist.smart.fields.limit')}
                  fullWidth
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <SelectInput
                  source="sort"
                  label="resources.playlist.smart.fields.sort"
                  choices={sortChoices.map((choice) => ({
                    ...choice,
                    name: translate(choice.name),
                  }))}
                  emptyText={translate('resources.playlist.smart.sort.default')}
                  emptyValue=""
                  fullWidth
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <SelectInput
                  source="order"
                  label="resources.playlist.smart.fields.order"
                  choices={orderChoices.map((choice) => ({
                    ...choice,
                    name: translate(choice.name),
                  }))}
                  emptyText={translate('resources.playlist.smart.order.default')}
                  emptyValue=""
                  fullWidth
                />
              </Grid>
            </Grid>
          )
        }
      </FormDataConsumer>
    </>
  )
}

export default SmartPlaylistFields
