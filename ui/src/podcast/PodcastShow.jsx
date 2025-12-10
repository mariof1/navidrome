import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
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
import { Title, useGetIdentity, useNotify, usePermissions, useRedirect, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { useParams } from 'react-router-dom'
import { setTrack } from '../actions'
import PodcastFormDialog from './PodcastFormDialog'
import HtmlDescription from './HtmlDescription'
import {
  deletePodcastChannel,
  getPodcastChannel,
  listPodcastEpisodes,
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
  const [dialogOpen, setDialogOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const canManage = useMemo(
    () => isAdmin || channel?.userId === identity?.id,
    [channel?.userId, identity?.id, isAdmin],
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

  const handleSave = async ({ rssUrl, isGlobal }) => {
    if (!channel) return
    const url = (rssUrl || '').trim() || channel?.rssUrl
    if (!url) return
    setSaving(true)
    try {
      await updatePodcastChannel(channel.id, { rssUrl: url, isGlobal })
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
      notify('ra.notification.deleted', { type: 'info' })
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
                  <div className={classes.episodeDetails}>
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
                  </div>
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
            </ListItem>
            <Divider variant="inset" component="li" />
          </React.Fragment>
        ))}
      </List>
    )
  }, [channel?.imageUrl, classes, episodes, handlePlay, translate])

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
                    <Tooltip title={translate('ra.action.delete')}>
                      <IconButton
                        size="small"
                        onClick={() => setDeleteDialogOpen(true)}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
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
        initialValue={{ rssUrl: channel.rssUrl || '', isGlobal: channel.isGlobal || false }}
        allowGlobal={canManage}
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
        <DialogTitle>{translate('ra.action.delete')}</DialogTitle>
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
            {translate('ra.action.delete')}
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  )
}

export default PodcastShow
