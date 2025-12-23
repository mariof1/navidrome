import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useDispatch } from 'react-redux'
import {
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  CardHeader,
  CardMedia,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  Toolbar,
  Tooltip,
  Typography,
} from '@material-ui/core'
import AddIcon from '@material-ui/icons/Add'
import DeleteIcon from '@material-ui/icons/Delete'
import EditIcon from '@material-ui/icons/Edit'
import RefreshIcon from '@material-ui/icons/Refresh'
import {
  Title,
  useGetIdentity,
  useNotify,
  usePermissions,
  useRedirect,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import PodcastFormDialog from './PodcastFormDialog'
import HtmlDescription from './HtmlDescription'
import {
  createPodcastChannel,
  deletePodcastChannel,
  getPodcastChannel,
  listPodcasts,
  listContinueListening,
  updatePodcastChannel,
} from './api'
import { setTrack } from '../actions'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(2),
    maxWidth: '100%',
    overflowX: 'hidden',
  },
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  content: {
    maxWidth: '100%',
    overflowX: 'hidden',
  },
  card: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
  },
  cardActionArea: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'stretch',
    height: '100%',
  },
  media: {
    height: 160,
    backgroundSize: 'cover',
    backgroundColor: theme.palette.action.hover,
    width: '100%',
  },
  cardContent: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
    flexGrow: 1,
    overflow: 'hidden',
  },
  description: {
    color: theme.palette.text.secondary,
  },
  chips: {
    display: 'flex',
    gap: theme.spacing(1),
    flexWrap: 'wrap',
  },
  actions: {
    display: 'flex',
    gap: theme.spacing(1),
  },
  addButton: {
    marginLeft: theme.spacing(1),
  },
}))

const PodcastList = () => {
  const classes = useStyles()
  const notify = useNotify()
  const translate = useTranslate()
  const redirect = useRedirect()
  const { permissions } = usePermissions()
  const { identity } = useGetIdentity()
  const isAdmin = permissions === 'admin'
  const dispatch = useDispatch()
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState(null)
  const [saving, setSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState(null)

  const initialDialogValue = useMemo(
    () => ({
      rssUrl: editingChannel?.rssUrl || editingChannel?.rssURL || '',
    }),
    [editingChannel?.rssUrl, editingChannel?.rssURL],
  )

  const canManageChannel = useCallback(
    (channel) => isAdmin || channel?.userId === identity?.userId,
    [identity?.userId, isAdmin],
  )

  const loadChannels = useCallback(async () => {
    setLoading(true)
    try {
      const data = await listPodcasts()
      setChannels(data || [])
    } catch (err) {
      notify(
        err?.message || translate('resources.podcast.notifications.loadError'),
        { type: 'warning' },
      )
    } finally {
      setLoading(false)
    }
  }, [notify, translate])

  useEffect(() => {
    loadChannels()
  }, [loadChannels])

  useEffect(() => {
    if (!dialogOpen || !editingChannel || editingChannel.rssUrl || editingChannel.rssURL) {
      return
    }

    let isActive = true

    getPodcastChannel(editingChannel.id)
      .then((channel) => {
        if (!isActive) return
        setEditingChannel((prev) =>
          prev?.id === channel.id ? { ...prev, ...channel } : prev,
        )
      })
      .catch((err) => {
        notify(
          err?.message || translate('resources.podcast.notifications.loadError'),
          { type: 'warning' },
        )
      })

    return () => {
      isActive = false
    }
  }, [dialogOpen, editingChannel, notify, translate])

  const handleSave = async ({ rssUrl }) => {
    setSaving(true)
    try {
      if (editingChannel) {
        await updatePodcastChannel(editingChannel.id, { rssUrl })
        notify('resources.podcast.notifications.updated', { type: 'info' })
      } else {
        await createPodcastChannel({ rssUrl })
        notify('resources.podcast.notifications.created', { type: 'info' })
      }
      setDialogOpen(false)
      setEditingChannel(null)
      loadChannels()
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
    if (!deleteTarget) return
    setSaving(true)
    try {
      await deletePodcastChannel(deleteTarget.id)
      notify('resources.podcast.notifications.deleted', { type: 'info' })
      loadChannels()
    } catch (err) {
      notify(
        err?.message || translate('resources.podcast.notifications.loadError'),
        { type: 'warning' },
      )
    } finally {
      setSaving(false)
      setDeleteTarget(null)
    }
  }

  const handleCardClick = useCallback(
    (id) => {
      redirect(`/podcast/${id}/show`)
    },
    [redirect],
  )

  const handleContinueListening = async () => {
    try {
      const items = await listContinueListening(1)
      if (!items?.length) {
        notify(translate('resources.podcast.messages.noContinue'), { type: 'info' })
        return
      }
      const item = items[0]
      const episode = item.episode
      const channel = item.channel
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
          resumePosition: item.position || 0,
        }),
      )
    } catch (err) {
      notify(err?.message || translate('resources.podcast.notifications.loadError'), {
        type: 'warning',
      })
    }
  }

  const content = useMemo(() => {
    if (loading) {
      return (
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      )
    }

    if (!channels.length) {
      return (
        <Box p={4} textAlign="center">
          <Typography variant="body1">
            {translate('resources.podcast.messages.empty')}
          </Typography>
        </Box>
      )
    }

    return (
      <Grid container spacing={2}>
        {channels.map((channel) => {
          const canManage = canManageChannel(channel)

          return (
            <Grid item xs={12} sm={6} md={4} key={channel.id}>
              <Card className={classes.card}>
                <CardActionArea
                  onClick={() => handleCardClick(channel.id)}
                  className={classes.cardActionArea}
                >
                  <CardHeader
                    title={channel.title}
                    subheader={channel.siteUrl || ''}
                    titleTypographyProps={{ variant: 'h6' }}
                    subheaderTypographyProps={{ style: { wordBreak: 'break-word' } }}
                    action={
                      canManage && (
                        <div className={classes.actions}>
                          <Tooltip title={translate('ra.action.edit')}>
                            <IconButton
                              size="small"
                              onClick={(e) => {
                                e.stopPropagation()
                                setEditingChannel(channel)
                                setDialogOpen(true)
                              }}
                            >
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip
                            title={translate('resources.podcast.actions.unsubscribe', {
                              _: 'Unsubscribe',
                            })}
                          >
                            <IconButton
                              size="small"
                              onClick={(e) => {
                                e.stopPropagation()
                                setDeleteTarget(channel)
                              }}
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </div>
                      )
                    }
                  />
                  <CardMedia
                    className={classes.media}
                    image={channel.imageUrl || ''}
                    title={channel.title}
                  />
                  <CardContent className={classes.cardContent}>
                    <HtmlDescription
                      value={channel.description}
                      className={classes.description}
                    />
                    <div className={classes.chips}>
                      {channel.lastRefreshedAt && (
                        <Chip
                          size="small"
                          label={`${translate('resources.podcast.fields.lastRefreshedAt')}: ${new Date(
                            channel.lastRefreshedAt,
                          ).toLocaleString()}`}
                        />
                      )}
                      {channel.episodeCount ? (
                        <Chip
                          size="small"
                          label={`${translate('resources.podcast.fields.episodeCount')}: ${channel.episodeCount}`}
                        />
                      ) : null}
                    </div>
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          )
        })}
      </Grid>
    )
  }, [canManageChannel, channels, classes, handleCardClick, loading, translate])

  return (
    <Card className={classes.root}>
      <Title title={`Navidrome - ${translate('resources.podcast.name', { smart_count: 2 })}`} />
      <Toolbar className={classes.toolbar}>
        <Typography variant="h6">
          {translate('resources.podcast.name', { smart_count: 2 })}
        </Typography>
        <div>
          <IconButton aria-label={translate('ra.action.refresh')} onClick={loadChannels}>
            <RefreshIcon />
          </IconButton>
          <Button color="default" variant="outlined" onClick={handleContinueListening}>
            {translate('resources.podcast.actions.continueListening')}
          </Button>
          <Button
            color="primary"
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => {
              setEditingChannel(null)
              setDialogOpen(true)
            }}
            className={classes.addButton}
          >
            {translate('resources.podcast.actions.add')}
          </Button>
        </div>
      </Toolbar>
      <Box p={2} className={classes.content}>
        {content}
      </Box>

      <PodcastFormDialog
        open={dialogOpen}
        title={
          editingChannel
            ? translate('ra.action.edit')
            : translate('resources.podcast.actions.add')
        }
        initialValue={initialDialogValue}
        editMode={!!editingChannel}
        saving={saving}
        onClose={() => {
          setDialogOpen(false)
          setEditingChannel(null)
        }}
        onSave={handleSave}
      />

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)} fullWidth maxWidth="xs">
        <DialogTitle>
          {translate('resources.podcast.actions.unsubscribe', { _: 'Unsubscribe' })}
        </DialogTitle>
        <DialogContent>
          <Typography>{translate('ra.message.are_you_sure')}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)} color="default">
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

export default PodcastList
