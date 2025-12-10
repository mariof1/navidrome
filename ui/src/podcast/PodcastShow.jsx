import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Button,
  Card,
  CardContent,
  Divider,
  Grid,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  makeStyles,
  Typography,
} from '@material-ui/core'
import LaunchIcon from '@material-ui/icons/Launch'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { Title, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { useParams } from 'react-router-dom'
import { setTrack } from '../actions'
import { getPodcastChannel, listPodcastEpisodes } from './api'

const useStyles = makeStyles((theme) => ({
  hero: {
    display: 'flex',
    gap: theme.spacing(3),
    alignItems: 'center',
  },
  cover: {
    width: theme.spacing(12),
    height: theme.spacing(12),
    borderRadius: theme.shape.borderRadius,
  },
  description: {
    marginTop: theme.spacing(1),
    maxWidth: 720,
  },
  episodesHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
}))

const PodcastShow = () => {
  const classes = useStyles()
  const notify = useNotify()
  const translate = useTranslate()
  const dispatch = useDispatch()
  const { id } = useParams()
  const [channel, setChannel] = useState()
  const [episodes, setEpisodes] = useState([])
  const [loading, setLoading] = useState(true)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const ch = await getPodcastChannel(id)
      setChannel(ch)
      const episodeList = ch?.episodes || (await listPodcastEpisodes(id))
      setEpisodes(episodeList || [])
    } catch (err) {
      notify(
        err?.message || translate('resources.podcast.notifications.loadError'),
        { type: 'warning' },
      )
    } finally {
      setLoading(false)
    }
  }, [id, notify, translate])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handlePlay = useCallback(
    (episode) => {
      dispatch(
        setTrack({
          id: episode.id,
          title: episode.title,
          artist: channel?.title,
          album: channel?.title,
          duration: episode.duration,
          cover: episode.imageUrl || channel?.imageUrl,
          streamUrl: episode.audioUrl,
          isRadio: true,
        }),
      )
    },
    [channel?.imageUrl, channel?.title, dispatch],
  )

  const episodesList = useMemo(() => {
    if (!episodes.length) {
      return (
        <Typography variant="body2">
          {translate('resources.podcast.messages.noEpisodes')}
        </Typography>
      )
    }

    return (
      <List>
        {episodes.map((episode) => (
          <React.Fragment key={episode.id}>
            <ListItem button onClick={() => handlePlay(episode)} alignItems="flex-start">
              <ListItemAvatar>
                <Avatar
                  variant="square"
                  src={episode.imageUrl || channel?.imageUrl}
                  alt={episode.title}
                >
                  <PlayArrowIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText
                primary={episode.title}
                secondary={
                  <>
                    <Typography component="span" variant="body2" color="textPrimary">
                      {episode.publishedAt
                        ? new Date(episode.publishedAt).toLocaleDateString()
                        : ''}
                    </Typography>
                    {episode.duration
                      ? ` • ${translate('resources.song.fields.duration')}: ${Math.round(
                          episode.duration / 60,
                        )}m`
                      : ''}
                    {episode.description ? ` • ${episode.description}` : ''}
                  </>
                }
              />
              <IconButton edge="end" onClick={() => handlePlay(episode)}>
                <PlayArrowIcon />
              </IconButton>
            </ListItem>
            <Divider variant="inset" component="li" />
          </React.Fragment>
        ))}
      </List>
    )
  }, [channel?.imageUrl, episodes, handlePlay, translate])

  if (loading) {
    return (
      <Card>
        <CardContent>
          <Typography variant="body2">
            {translate('resources.podcast.messages.loading')}
          </Typography>
        </CardContent>
      </Card>
    )
  }

  if (!channel) {
    return null
  }

  return (
    <Card>
      <Title title={`Navidrome - ${channel.title}`} />
      <CardContent>
        <Grid container spacing={3}>
          <Grid item xs={12} className={classes.hero}>
            <Avatar
              variant="square"
              src={channel.imageUrl}
              alt={channel.title}
              className={classes.cover}
            />
            <div>
              <Typography variant="h5">{channel.title}</Typography>
              {channel.siteUrl && (
                <Button
                  color="primary"
                  endIcon={<LaunchIcon />}
                  href={channel.siteUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  {translate('resources.podcast.actions.visitSite')}
                </Button>
              )}
              {channel.description && (
                <Typography variant="body2" className={classes.description}>
                  {channel.description}
                </Typography>
              )}
            </div>
          </Grid>
          <Grid item xs={12}>
            <div className={classes.episodesHeader}>
              <Typography variant="h6">
                {translate('resources.podcast.fields.episodes')}
              </Typography>
            </div>
            {episodesList}
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  )
}

export default PodcastShow
