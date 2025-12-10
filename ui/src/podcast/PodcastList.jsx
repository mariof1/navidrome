import React, { useCallback, useEffect, useMemo, useState } from 'react'
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
  FormControlLabel,
  Grid,
  IconButton,
  makeStyles,
  TextField,
  Toolbar,
  Typography,
  Checkbox,
} from '@material-ui/core'
import AddIcon from '@material-ui/icons/Add'
import RefreshIcon from '@material-ui/icons/Refresh'
import RssFeedIcon from '@material-ui/icons/RssFeed'
import { Title, useNotify, usePermissions, useRedirect, useTranslate } from 'react-admin'
import { createPodcastChannel, listPodcasts } from './api'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(2),
  },
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  media: {
    height: 140,
    backgroundSize: 'cover',
    backgroundColor: theme.palette.action.hover,
  },
  cardContent: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
    minHeight: 160,
  },
  chips: {
    display: 'flex',
    gap: theme.spacing(1),
    flexWrap: 'wrap',
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
  const isAdmin = permissions === 'admin'
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [rssUrl, setRssUrl] = useState('')
  const [isGlobal, setIsGlobal] = useState(false)
  const [saving, setSaving] = useState(false)

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

  const handleCreate = async () => {
    setSaving(true)
    try {
      await createPodcastChannel({ rssUrl, isGlobal })
      notify('resources.podcast.notifications.created', { type: 'info' })
      setAddDialogOpen(false)
      setRssUrl('')
      setIsGlobal(false)
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

  const handleCardClick = (id) => {
    redirect(`/podcast/${id}/show`)
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
        {channels.map((channel) => (
          <Grid item xs={12} sm={6} md={4} key={channel.id}>
            <Card>
              <CardActionArea onClick={() => handleCardClick(channel.id)}>
                <CardHeader
                  title={channel.title}
                  subheader={channel.siteUrl || ''}
                  titleTypographyProps={{ variant: 'h6' }}
                />
                <CardMedia
                  className={classes.media}
                  image={channel.imageUrl || ''}
                  title={channel.title}
                />
                <CardContent className={classes.cardContent}>
                  {channel.description && (
                    <Typography variant="body2" color="textSecondary" noWrap>
                      {channel.description}
                    </Typography>
                  )}
                  <div className={classes.chips}>
                    {channel.isGlobal && (
                      <Chip
                        color="primary"
                        size="small"
                        label={translate('resources.podcast.labels.shared')}
                      />
                    )}
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
        ))}
      </Grid>
    )
  }, [channels, classes.cardContent, classes.chips, classes.media, loading, translate])

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
          <Button
            color="primary"
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddDialogOpen(true)}
            className={classes.addButton}
          >
            {translate('resources.podcast.actions.add')}
          </Button>
        </div>
      </Toolbar>
      <Box p={2}>{content}</Box>

      <Dialog open={addDialogOpen} onClose={() => setAddDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>{translate('resources.podcast.actions.add')}</DialogTitle>
        <DialogContent>
          <TextField
            label={translate('resources.podcast.fields.rssUrl')}
            value={rssUrl}
            fullWidth
            onChange={(e) => setRssUrl(e.target.value)}
            margin="normal"
            autoFocus
          />
          <FormControlLabel
            control={
              <Checkbox
                color="primary"
                checked={isGlobal}
                onChange={(e) => setIsGlobal(e.target.checked)}
                disabled={!isAdmin}
              />
            }
            label={translate('resources.podcast.fields.isGlobal')}
          />
          {!isAdmin && (
            <Typography variant="caption" color="textSecondary">
              {translate('resources.podcast.messages.adminOnly')}
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddDialogOpen(false)} color="default">
            {translate('ra.action.cancel')}
          </Button>
          <Button
            onClick={handleCreate}
            color="primary"
            variant="contained"
            startIcon={<RssFeedIcon />}
            disabled={!rssUrl || saving}
          >
            {translate('resources.podcast.actions.save')}
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  )
}

export default PodcastList
