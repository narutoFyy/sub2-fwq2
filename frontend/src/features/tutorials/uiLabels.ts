import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'

/**
 * 教程和引导里提到的界面文字，一律从站点自己的语言包取，
 * 这样改了按钮文案（比如「导入到 CCS」）教程会跟着变，不会再写错。
 */
export function useUiLabels() {
  const { t } = useI18n()
  const app = useAppStore()
  return computed(() => {
    const shop = (app.cachedPublicSettings?.custom_menu_items ?? [])
      .filter((item) => item.visibility === 'user')
      .sort((a, b) => a.sort_order - b.sort_order)[0]
    return {
      apiKeys: t('nav.apiKeys'),
      redeem: t('nav.redeem'),
      usage: t('nav.usage'),
      createKey: t('keys.createKey'),
      importCcs: t('keys.importToCcSwitch'),
      provider: t('keys.providerLabel'),
      group: t('keys.groupLabel'),
      name: t('keys.nameLabel'),
      create: t('common.create'),
      redeemBtn: t('redeem.redeemButton'),
      /** 侧栏里通往购买页的入口名称：线上是自定义菜单「购买套餐」。 */
      buy: shop?.label || t('nav.recharge')
    }
  })
}

export type UiLabels = ReturnType<typeof useUiLabels>['value']
