import React from 'react'
import { Grid, Typography } from '@material-ui/core'
import {
  AutocompleteArrayInput,
  BooleanInput,
  FormDataConsumer,
  NumberInput,
  ReferenceArrayInput,
  SelectInput,
  useTranslate,
} from 'react-admin'

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

const SmartPlaylistFields = () => {
  const translate = useTranslate()

  return (
    <>
      <BooleanInput source="smart" label="resources.playlist.fields.smart" />
      <FormDataConsumer>
        {({ formData }) =>
          formData.smart && (
            <Grid container spacing={2}>
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
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="maxDuration"
                  label={translate('resources.playlist.smart.fields.maxDuration')}
                  helperText={translate('resources.playlist.smart.help.duration')}
                />
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="minPlayCount"
                  label={translate('resources.playlist.smart.fields.minPlayCount')}
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <NumberInput
                  source="maxPlayCount"
                  label={translate('resources.playlist.smart.fields.maxPlayCount')}
                />
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <BooleanInput
                  source="includeAllUsersPlayCount"
                  label={translate('resources.playlist.smart.fields.playCountAllUsers')}
                />
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="includeArtists"
                  reference="artist"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.includeArtist"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="excludeArtists"
                  reference="artist"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.excludeArtist"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="includeAlbums"
                  reference="album"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.includeAlbum"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="excludeAlbums"
                  reference="album"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.excludeAlbum"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
              </Grid>

              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="includeGenres"
                  reference="genre"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.includeGenre"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <ReferenceArrayInput
                  source="excludeGenres"
                  reference="genre"
                  sort={{ field: 'name', order: 'ASC' }}
                  filterToQuery={(searchText) => ({ name: [searchText] })}
                  perPage={0}
                  label="resources.playlist.smart.fields.excludeGenre"
                >
                  <AutocompleteArrayInput optionText="name" optionValue="name" />
                </ReferenceArrayInput>
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
