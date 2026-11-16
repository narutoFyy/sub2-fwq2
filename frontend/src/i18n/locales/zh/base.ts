// 公开页面（首页、登录、教程等）只需要这份基础文案。
// 后台文案体积占整份语言包的大半，放在 adminPack.ts 里，进入登录后的页面时再加载。
import mcp from './mcp'
import landing from './landing'
import activity from './activity'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import chat from './chat'
import misc from './misc'

export default {
  tutorials: { title: '接入教程' },
  mcp,
  ...landing,
  ...activity,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  ...chat,
  ...misc,
}
