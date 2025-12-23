import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  Tooltip,
  Typography,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import LaunchIcon from '@material-ui/icons/Launch'
import DeleteIcon from '@material-ui/icons/Delete'
import EditIcon from '@material-ui/icons/Edit'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import VisibilityIcon from '@material-ui/icons/Visibility'
import VisibilityOffIcon from '@material-ui/icons/VisibilityOff'
import { Title, useGetIdentity, useNotify, usePermissions, useRedirect, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { useParams } from 'react-router-dom'
import { setTrack } from '../actions'
import PodcastFormDialog from './PodcastFormDialog'
import HtmlDescription from './HtmlDescription'
import {
  deletePodcastChannel,
  getEpisodeProgress,
  getPodcastChannel,
  listPodcastEpisodes,
  setEpisodeWatched,
  updatePodcastChannel,
} from './api'

const useStyles = makeStyles((theme) => ({
  root: {
    maxWidth: '100%',
    overflowX: 'hidden',
  },
  hero: {
    display: 'flex',
    gap: theme.spacing(3),
    alignItems: 'center',
    flexWrap: 'wrap',
    width: '100%',
  },
  cover: {
    width: theme.spacing(12),
    height: theme.spacing(12),
    borderRadius: theme.shape.borderRadius,
    flexShrink: 0,
  },
  description: {
    marginTop: theme.spacing(1),
    maxWidth: 720,
  },
  episodesHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    flexWrap: 'wrap',
    gap: theme.spacing(1),
  },
  headerActions: {
    display: 'flex',
    gap: theme.spacing(1),
    alignItems: 'center',
  },
  episodeItem: {
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'flex-start',
  },
  episodeAvatar: {
    width: theme.spacing(8),
    height: theme.spacing(8),
    flexShrink: 0,
  },
  episodeText: {
    minWidth: 0,
    flex: '1 1 200px',
  },
  episodeDetails: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(0.5),
    marginTop: theme.spacing(0.5),
  },
  list: {
    width: '100%',
  },
}))

const PodcastShow = () => {
  const classes = useStyles()
  const notify = useNotify()
  const translate = useTranslate()
  const redirect = useRedirect()
  const { permissions } = usePermissions()
  const { identity } = useGetIdentity()
  const isAdmin = permissions === 'admin'
  const dispatch = useDispatch()
  const { id } = useParams()
  const [channel, setChannel] = useState()
  const [episodes, setEpisodes] = useState([])
  const [loading, setLoading] = useState(true)
  const [expandedEpisodes, setExpandedEpisodes] = useState({})
  const [dialogOpen, setDialogOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const canManage = useMemo(
    () => isAdmin || channel?.userId === identity?.userId,
    [channel?.userId, identity?.userId, isAdmin],
  )

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

  const handleSave = async ({ rssUrl }) => {
    if (!channel) return
    const url = (rssUrl || '').trim() || channel?.rssUrl || channel?.rssURL
    if (!url) return
    setSaving(true)
    try {
      await updatePodcastChannel(channel.id, { rssUrl: url })
      notify('ra.notification.updated', { type: 'info' })
      setDialogOpen(false)
      loadData()
    } catch (err) {
      notify(
        err?.message || translate('resources.podcast.notifications.createError'),
        { type: 'warning' },
      )
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!channel) return
    setSaving(true)
    try {
      await deletePodcastChannel(channel.id)
      notify('resources.podcast.notifications.deleted', { type: 'info' })
      redirect('/podcast')
    } catch (err) {
      notify(
        err?.message || translate('resources.podcast.notifications.loadError'),
        { type: 'warning' },
      )
    } finally {
      setSaving(false)
      setDeleteDialogOpen(false)
    }
  }

  const handlePlay = useCallback(
    async (episode) => {
      let resumePosition = 0
      try {
        const progress = await getEpisodeProgress(channel?.id, episode.id)
        resumePosition = progress?.position || 0
      } catch (_) {
        // Ignore progress lookup errors
      }

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
          isPodcast: true,
          channelId: channel?.id,
          resumePosition,
        }),
      )
    },
    [channel?.id, channel?.imageUrl, channel?.title, dispatch],
  )

  const toggleWatched = useCallback(
    async (episode, watched) => {
      setEpisodes((prev) =>
        prev.map((item) =>
          item.id === episode.id
            ? {
                ...item,
                watched,
              }
            : item,
        ),
      )
      try {
        await setEpisodeWatched(channel.id, episode.id, watched)
      } catch (err) {
        setEpisodes((prev) =>
          prev.map((item) =>
            item.id === episode.id
              ? {
                  ...item,
                  watched: !watched,
                }
              : item,
          ),
        )
        notify(err?.message || translate('resources.podcast.notifications.loadError'), {
          type: 'warning',
        })
      }
    },
    [channel?.id, notify, translate],
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
      <List className={classes.list}>
        {episodes.map((episode) => (
          <React.Fragment key={episode.id}>
            <ListItem
              button
              onClick={() => handlePlay(episode)}
              alignItems="flex-start"
              className={classes.episodeItem}
            >
              <ListItemAvatar>
                <Avatar
                  variant="square"
                  src={episode.imageUrl || channel?.imageUrl}
                  alt={episode.title}
                  className={classes.episodeAvatar}
                >
                  <PlayArrowIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText
                className={classes.episodeText}
                primary={episode.title}
                secondary={
                  <Button
                    size="small"
                    color="primary"
                    onClick={(e) => {
                      e.stopPropagation()
                      setExpandedEpisodes((prev) => ({
                        ...prev,
                        [episode.id]: !prev[episode.id],
                      }))
                    }}
                  >
                    {expandedEpisodes[episode.id]
                      ? translate('resources.podcast.actions.hideDetails', {
                          _: 'Hide details',
                        })
                      : translate('resources.podcast.actions.viewDetails', {
                          _: 'View details',
                        })}
                  </Button>
                }
                secondaryTypographyProps={{ component: 'div' }}
              />
              <IconButton
                edge="end"
                onClick={(e) => {
                  e.stopPropagation()
                  handlePlay(episode)
                }}
              >
                <PlayArrowIcon />
              </IconButton>
              <Tooltip
                title={
                  episode.watched
                    ? translate('resources.podcast.actions.markUnwatched')
                    : translate('resources.podcast.actions.markWatched')
                }
              >
                <IconButton
                  edge="end"
                  onClick={(e) => {
                    e.stopPropagation()
                    toggleWatched(episode, !episode.watched)
                  }}
                >
                  {episode.watched ? <VisibilityIcon color="primary" /> : <VisibilityOffIcon />}
                </IconButton>
              </Tooltip>
            </ListItem>
            <Collapse in={expandedEpisodes[episode.id]} timeout="auto" unmountOnExit>
              <Box pl={11} pr={2} pb={2} className={classes.episodeDetails}>
                <Typography component="div" variant="body2" color="textPrimary">
                  {episode.publishedAt
                    ? new Date(episode.publishedAt).toLocaleDateString()
                    : ''}
                  {episode.duration
                    ? ` • ${translate('resources.song.fields.duration')}: ${Math.round(
                        episode.duration / 60,
                      )}m`
                    : ''}
                </Typography>
                <HtmlDescription value={episode.description} />
              </Box>
            </Collapse>
            <Divider variant="inset" component="li" />
          </React.Fragment>
        ))}
      </List>
    )
  }, [channel?.imageUrl, classes, episodes, expandedEpisodes, handlePlay, toggleWatched, translate])

  if (loading) {
    return (
      <Card className={classes.root}>
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
    <Card className={classes.root}>
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
              <Box
                display="flex"
                alignItems="center"
                justifyContent="space-between"
                flexWrap="wrap"
                style={{ gap: 8 }}
              >
                <Typography variant="h5">{channel.title}</Typography>
                {canManage && (
                  <div className={classes.headerActions}>
                    <Tooltip title={translate('ra.action.edit')}>
                      <IconButton size="small" onClick={() => setDialogOpen(true)}>
                        <EditIcon />
                      </IconButton>
                    </Tooltip>
                    <Button
                      size="small"
                      variant="outlined"
                      color="secondary"
                      startIcon={<DeleteIcon />}
                      onClick={() => setDeleteDialogOpen(true)}
                    >
                      {translate('resources.podcast.actions.unsubscribe', {
                        _: 'Unsubscribe',
                      })}
                    </Button>
                  </div>
                )}
              </Box>
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
              <HtmlDescription value={channel.description} className={classes.description} />
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

      <PodcastFormDialog
        open={dialogOpen}
        title={translate('ra.action.edit')}
        initialValue={{ rssUrl: channel.rssUrl || channel.rssURL || '' }}
        editMode
        saving={saving}
        onClose={() => setDialogOpen(false)}
        onSave={handleSave}
      />

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>
          {translate('resources.podcast.actions.unsubscribe', { _: 'Unsubscribe' })}
        </DialogTitle>
        <DialogContent>
          <Typography>{translate('ra.message.are_you_sure')}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)} color="default">
            {translate('ra.action.cancel')}
          </Button>
          <Button
            onClick={handleDelete}
            color="secondary"
            variant="contained"
            startIcon={<DeleteIcon />}
            disabled={saving}
          >
            {translate('resources.podcast.actions.unsubscribe', { _: 'Unsubscribe' })}
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  )
}

export default PodcastShow
