import React, { useEffect, useState } from 'react'
import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  TextField,
  Typography,
} from '@material-ui/core'
import RssFeedIcon from '@material-ui/icons/RssFeed'
import { useTranslate } from 'react-admin'

const PodcastFormDialog = ({
  open,
  onClose,
  onSave,
  initialValue = {},
  allowGlobal = true,
  saving = false,
  title,
}) => {
  const translate = useTranslate()
  const [rssUrl, setRssUrl] = useState(initialValue.rssUrl || initialValue.rssURL || '')
  const [isGlobal, setIsGlobal] = useState(initialValue.isGlobal || false)

  useEffect(() => {
    setRssUrl(initialValue.rssUrl || initialValue.rssURL || '')
    setIsGlobal(initialValue.isGlobal || false)
  }, [initialValue, open])

  const handleSubmit = () => onSave({ rssUrl, isGlobal })

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <TextField
          label={translate('resources.podcast.fields.rssUrl')}
          value={rssUrl}
          fullWidth
          onChange={(e) => setRssUrl(e.target.value)}
          margin="normal"
          autoFocus
        />
        {allowGlobal ? (
          <FormControlLabel
            control={
              <Checkbox
                color="primary"
                checked={isGlobal}
                onChange={(e) => setIsGlobal(e.target.checked)}
              />
            }
            label={translate('resources.podcast.fields.isGlobal')}
          />
        ) : (
          <Typography variant="caption" color="textSecondary">
            {translate('resources.podcast.messages.adminOnly')}
          </Typography>
        )}
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
          {translate('resources.podcast.actions.save')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default PodcastFormDialog
