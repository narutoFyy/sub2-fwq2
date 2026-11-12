import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import activity from './activity'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import chat from './chat'
import admin from './admin'
import misc from './misc'

export default {
  tutorials: { title: 'Setup guides' },
  adminAccessKeys,
  mcp,
  ...landing,
  ...activity,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  ...chat,
  admin,
  ...misc,
}
