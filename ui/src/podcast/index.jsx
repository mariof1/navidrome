import React from 'react'
import MicIcon from '@material-ui/icons/Mic'
import MicNoneIcon from '@material-ui/icons/MicNone'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import PodcastList from './PodcastList'
import PodcastShow from './PodcastShow'

export default {
  list: PodcastList,
  show: PodcastShow,
  icon: (
    <DynamicMenuIcon path={'podcast'} icon={MicNoneIcon} activeIcon={MicIcon} />
  ),
}
