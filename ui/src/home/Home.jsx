import React, { useEffect, useMemo, useState } from 'react'
import { useTranslate, Loading, Title, useDataProvider } from 'react-admin'
import { Typography, makeStyles, useMediaQuery, IconButton } from '@material-ui/core'
import { useSelector } from 'react-redux'
import { useDispatch } from 'react-redux'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import subsonic from '../subsonic'
import { getHomeRecommendations } from './api'
import { playTracks, shuffleTracks } from '../actions'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      padding: theme.spacing(2),
      maxWidth: 1400,
      margin: '0 auto',
      paddingBottom: (props) => (props.addPadding ? '80px' : theme.spacing(2)),
      boxSizing: 'border-box',
      width: '100%',
      minWidth: 0,
      overflowX: 'hidden',
    },
    section: {
      marginTop: theme.spacing(3),
    },
    header: {
      display: 'flex',
      alignItems: 'baseline',
      justifyContent: 'space-between',
      flexWrap: 'wrap',
      gap: theme.spacing(1),
      marginBottom: theme.spacing(1),
      minWidth: 0,
      '& > *': {
        minWidth: 0,
      },
    },
    headerActions: {
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(0.5),
      flexShrink: 0,
    },
    row: {
      display: 'grid',
      gap: theme.spacing(2),
      overflowX: 'hidden',
      gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))',
      paddingBottom: theme.spacing(1),
      [theme.breakpoints.down('xs')]: {
        gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
      },
    },
    bucketCard: {
      width: '100%',
      minWidth: 0,
      borderRadius: theme.shape.borderRadius,
      overflow: 'hidden',
      background: theme.palette.background.paper,
      cursor: 'pointer',
      userSelect: 'none',
    },
    bucketArtGrid: {
      display: 'grid',
      gridTemplateColumns: 'repeat(2, 1fr)',
      gridTemplateRows: 'repeat(2, 1fr)',
      width: '100%',
      aspectRatio: '1 / 1',
      background: theme.palette.background.default,
    },
    bucketArt: {
      width: '100%',
      height: '100%',
      objectFit: 'cover',
      display: 'block',
    },
    bucketMeta: {
      padding: theme.spacing(1),
      minWidth: 0,
    },
    bucketName: {
      fontSize: 14,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
    bucketSubtitle: {
      fontSize: 12,
      opacity: 0.8,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
  }),
  { name: 'NDHome' },
)

const BucketRow = ({ title, items, loading }) => {
  const classes = useStyles()
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()

  if (loading) {
    return (
      <div className={classes.section}>
        <div className={classes.header}>
          <Typography variant="h6">{title}</Typography>
        </div>
        <Loading loadingPrimary="ra.page.loading" />
      </div>
    )
  }

  const albums = items || []
  if (albums.length === 0) {
    return null
  }

  const playBucket = async ({ shuffle } = {}) => {
    // Build a queue of songs from the bucket's albums.
    // Keep it lightweight by limiting to 500 total songs.
    const songs = {}
    const songIds = []
    for (const a of albums) {
      const res = await dataProvider.getList('song', {
        pagination: { page: 1, perPage: 200 },
        sort: { field: 'album', order: 'ASC' },
        filter: { album_id: a.id, missing: false },
      })
      res.data.forEach((s) => {
        if (!songs[s.id]) {
          songs[s.id] = s
          songIds.push(s.id)
        }
      })
      if (songIds.length >= 500) break
    }
    if (shuffle) {
      dispatch(shuffleTracks(songs, songIds))
    } else {
      dispatch(playTracks(songs, songIds))
    }
  }

  return (
    <div className={classes.section}>
      <div className={classes.header}>
        <Typography variant="h6">{title}</Typography>
        <div className={classes.headerActions}>
          <IconButton
            aria-label="play"
            size="small"
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              playBucket({ shuffle: false })
            }}
          >
            <PlayArrowIcon fontSize="small" />
          </IconButton>
          <IconButton
            aria-label="shuffle"
            size="small"
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              playBucket({ shuffle: true })
            }}
          >
            <ShuffleIcon fontSize="small" />
          </IconButton>
        </div>
      </div>
      <div className={classes.row}>
        {albums.map((record) => {
          const art = albums
            .slice(0, 4)
            .map((a) => subsonic.getCoverArtUrl(a, 300, true))
          while (art.length < 4) art.push(art[0])

          return (
            <div
              key={record.id}
              className={classes.bucketCard}
              role="button"
              tabIndex={0}
              onClick={() => playBucket({ shuffle: false })}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  playBucket({ shuffle: false })
                }
              }}
            >
              <div className={classes.bucketArtGrid}>
                {art.slice(0, 4).map((src, idx) => (
                  <img
                    key={idx}
                    className={classes.bucketArt}
                    src={src}
                    alt=""
                    loading="lazy"
                  />
                ))}
              </div>
              <div className={classes.bucketMeta}>
                <Typography className={classes.bucketName}>{title}</Typography>
                <Typography className={classes.bucketSubtitle}>
                  {albums.length} albums
                </Typography>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

const Home = () => {
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue.length > 0 })
  const translate = useTranslate()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'), {
    noSsr: true,
  })

  const perPage = isDesktop ? 12 : 8

  const seed = useMemo(() => Math.random().toString(36).slice(2), [])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [sections, setSections] = useState([])

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(null)
    getHomeRecommendations({ limit: perPage, seed })
      .then((resp) => {
        if (!active) return
        setSections(resp?.sections || [])
      })
      .catch((e) => {
        if (!active) return
        setError(e)
        setSections([])
      })
      .finally(() => {
        if (!active) return
        setLoading(false)
      })

    return () => {
      active = false
    }
  }, [perPage, seed])

  const titleFallback = (id) => {
    switch (id) {
      case 'dailyMix1':
        return 'Daily mix 1'
      case 'dailyMix2':
        return 'Daily mix 2'
      case 'dailyMix3':
        return 'Daily mix 3'
      case 'inspiredBy':
        return 'Inspired by you'
      case 'recentlyPlayed':
        return 'Recently played'
      case 'starred':
        return 'Starred'
      case 'recentlyAdded':
        return 'Recently added'
      case 'mostPlayed':
        return 'Most played'
      case 'onRepeat':
        return 'On repeat'
      case 'rediscover':
        return 'Rediscover'
      case 'discoverFresh':
        return 'Discover fresh'
      case 'random':
        return 'Random'
      default:
        return id
    }
  }

  return (
    <div className={classes.root}>
      <Title title={translate('menu.home', { _: 'Home' })} />

      {loading && <Loading loadingPrimary="ra.page.loading" />}

      {error && !loading && (
        <Typography color="error" variant="body2">
          {String(error)}
        </Typography>
      )}

      {sections
        .filter((s) => s?.resource === 'album')
        .map((s) => (
          <BucketRow
            key={s.id}
            title={translate(`resources.album.lists.${s.id}`, {
              smart_count: 2,
              _: titleFallback(s.id),
            })}
            items={s.items}
            loading={loading}
          />
        ))}
    </div>
  )
}

export default Home
