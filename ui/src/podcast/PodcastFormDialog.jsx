import React, { useEffect, useState } from 'react'
import {
  Avatar,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
  TextField,
  Typography,
} from '@material-ui/core'
import RssFeedIcon from '@material-ui/icons/RssFeed'
import { useTranslate } from 'react-admin'
import { searchApplePodcasts } from './api'

const PodcastFormDialog = ({
  open,
  onClose,
  onSave,
  initialValue = {},
  saving = false,
  title,
}) => {
  const translate = useTranslate()
  const [rssUrl, setRssUrl] = useState(initialValue.rssUrl || initialValue.rssURL || '')
  const [searchTerm, setSearchTerm] = useState('')
  const [searching, setSearching] = useState(false)
  const [results, setResults] = useState([])
  const [searchError, setSearchError] = useState('')

  const isEditMode = !!(initialValue.rssUrl || initialValue.rssURL)

  useEffect(() => {
    setRssUrl(initialValue.rssUrl || initialValue.rssURL || '')
    setSearchTerm('')
    setResults([])
    setSearchError('')
  }, [initialValue.rssURL, initialValue.rssUrl, open])

  const handleSubmit = () => onSave({ rssUrl })

  const handleSearch = async () => {
    const term = (searchTerm || '').trim()
    if (!term) return
    setSearching(true)
    setSearchError('')
    try {
      const res = await searchApplePodcasts(term)
      const rows = Array.isArray(res) ? res : []
      const seen = new Set()
      const deduped = []
      for (const row of rows) {
        if (!row) continue
        const feedUrl = (row.feedUrl || '').trim()
        const key = (feedUrl || `${row.title || ''}::${row.author || ''}`)
          .trim()
          .toLowerCase()
        if (!key || seen.has(key)) continue
        seen.add(key)
        deduped.push(row)
      }
      setResults(deduped)
    } catch (e) {
      setResults([])
      setSearchError(e?.message || translate('resources.podcast.notifications.loadError'))
    } finally {
      setSearching(false)
    }
  }

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        {!isEditMode ? (
          <>
            <TextField
              label={translate('resources.podcast.fields.searchApple')}
              value={searchTerm}
              fullWidth
              onChange={(e) => setSearchTerm(e.target.value)}
              margin="normal"
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  handleSearch()
                }
              }}
            />
            <Button
              onClick={handleSearch}
              color="default"
              variant="outlined"
              disabled={searching || !(searchTerm || '').trim()}
            >
              {translate('resources.podcast.actions.search')}
            </Button>

            {searching ? (
              <Typography variant="body2" style={{ marginTop: 12 }}>
                <CircularProgress size={18} style={{ marginRight: 8 }} />
                {translate('resources.podcast.messages.searching')}
              </Typography>
            ) : null}

            {!searching && searchError ? (
              <Typography
                variant="caption"
                color="error"
                style={{ display: 'block', marginTop: 8 }}
              >
                {searchError}
              </Typography>
            ) : null}

            {!searching && results?.length ? (
              <List dense style={{ maxHeight: 240, overflowY: 'auto' }}>
                {results.map((r) => (
                  <ListItem
                    key={`${r.feedUrl}-${r.title}`}
                    button
                    onClick={() => onSave({ rssUrl: r.feedUrl })}
                  >
                    <ListItemAvatar>
                      <Avatar variant="square" src={r.imageUrl || ''} />
                    </ListItemAvatar>
                    <ListItemText primary={r.title} secondary={r.author || r.siteUrl || ''} />
                  </ListItem>
                ))}
              </List>
            ) : null}

            {!searching && (searchTerm || '').trim() && results?.length === 0 && !searchError ? (
              <Typography
                variant="caption"
                color="textSecondary"
                style={{ display: 'block', marginTop: 8 }}
              >
                {translate('resources.podcast.messages.noSearchResults')}
              </Typography>
            ) : null}
          </>
        ) : null}

        <TextField
          label={translate('resources.podcast.fields.rssUrl')}
          value={rssUrl}
          fullWidth
          onChange={(e) => setRssUrl(e.target.value)}
          margin="normal"
          autoFocus
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color="default">
          {translate('ra.action.cancel')}
        </Button>
        <Button
          onClick={handleSubmit}
          color="primary"
          variant="contained"
          startIcon={<RssFeedIcon />}
          disabled={!rssUrl || saving}
        >
          {isEditMode ? translate('ra.action.save') : translate('resources.podcast.actions.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default PodcastFormDialog
