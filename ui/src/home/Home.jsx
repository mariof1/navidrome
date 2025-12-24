import React, { useEffect, useMemo, useState } from 'react'
import { useTranslate, linkToRecord, Loading, Title } from 'react-admin'
import { Link } from 'react-router-dom'
import { Typography, makeStyles, useMediaQuery } from '@material-ui/core'
import { useSelector } from 'react-redux'
import subsonic from '../subsonic'
import { getHomeRecommendations } from './api'

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
    card: {
      width: '100%',
      textDecoration: 'none',
      color: 'inherit',
      minWidth: 0,
    },
    cover: {
      width: '100%',
      aspectRatio: '1 / 1',
      objectFit: 'cover',
      borderRadius: theme.shape.borderRadius,
      display: 'block',
    },
    title: {
      marginTop: theme.spacing(1),
      fontSize: 14,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
    subtitle: {
      fontSize: 12,
      opacity: 0.8,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
  }),
  { name: 'NDHome' },
)

const AlbumRow = ({ title, to, items, loading }) => {
  const classes = useStyles()

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

  return (
    <div className={classes.section}>
      <div className={classes.header}>
        <Typography variant="h6">{title}</Typography>
        {to && (
          <Typography variant="body2" component={Link} to={to}>
            See all
          </Typography>
        )}
      </div>
      <div className={classes.row}>
        {albums.map((record) => (
          <Link
            key={record.id}
            className={classes.card}
            to={linkToRecord('/album', record.id, 'show')}
          >
            <img
              className={classes.cover}
              src={subsonic.getCoverArtUrl(record, 300, true)}
              alt={record.name}
              loading="lazy"
            />
            <Typography className={classes.title}>{record.name}</Typography>
            <Typography className={classes.subtitle}>
              {record.albumArtist}
            </Typography>
          </Link>
        ))}
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
      case 'recentlyPlayed':
        return 'Recently played'
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
          <AlbumRow
            key={s.id}
            title={translate(`resources.album.lists.${s.id}`, {
              smart_count: 2,
              _: titleFallback(s.id),
            })}
            to={s.to}
            items={s.items}
            loading={loading}
          />
        ))}
    </div>
  )
}

export default Home
