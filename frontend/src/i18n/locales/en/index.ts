// 完整语言包（测试与键完整性检查使用）；运行时按 base / adminPack 分开加载。
import base from './base'
import adminPack from './adminPack'

export default {
  ...base,
  ...adminPack,
}
