<template>
  <BaseDialog
    :show="show"
    :title="t('affiliate.poster.title')"
    width="narrow"
    :close-on-click-outside="true"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- Poster preview area -->
      <div
        class="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/70"
      >
        <div v-if="generating" class="flex flex-col items-center gap-3 py-12 text-sm text-gray-500 dark:text-dark-400">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
          <span>{{ t('affiliate.poster.generating') }}</span>
        </div>

        <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 py-10 text-center">
          <p class="text-sm text-red-600 dark:text-red-400">{{ errorMessage }}</p>
          <button type="button" class="btn btn-secondary btn-sm" @click="renderPoster">
            {{ t('affiliate.poster.retry') }}
          </button>
        </div>

        <template v-else-if="poster">
          <img
            :src="poster.dataUrl"
            :alt="t('affiliate.poster.previewAlt')"
            class="mx-auto w-full max-w-[300px] rounded-2xl shadow-lg ring-1 ring-black/10 dark:ring-white/10"
          />
          <p class="mt-3 text-center text-xs text-gray-500 dark:text-dark-400">
            {{ t('affiliate.poster.mobileHint') }}
          </p>
        </template>
      </div>

      <!-- Actions -->
      <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <button
          type="button"
          class="btn btn-primary btn-sm w-full"
          :disabled="!poster || generating"
          @click="downloadPoster"
        >
          <Icon name="download" size="sm" />
          <span>{{ t('affiliate.poster.download') }}</span>
        </button>

        <button
          type="button"
          class="btn btn-secondary btn-sm w-full"
          :disabled="!poster || generating || copyingImage"
          @click="copyPosterImage"
        >
          <Icon name="copy" size="sm" />
          <span>{{ t('affiliate.poster.copyImage') }}</span>
        </button>

        <button
          type="button"
          class="btn btn-secondary btn-sm w-full"
          :disabled="!normalizedCode"
          @click="copyShareText"
        >
          <Icon name="clipboard" size="sm" />
          <span>{{ t('affiliate.poster.copyCaption') }}</span>
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getActivePinia } from 'pinia'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { buildInviteLink, buildShareCopy, copyTextToClipboard, isZhLocale } from './helpers'
import { generateAffiliatePoster, type GeneratedAffiliatePoster } from './poster'

const props = withDefaults(
  defineProps<{
    show: boolean
    affCode: string
    inviteLink?: string
  }>(),
  {
    inviteLink: '',
  },
)

const emit = defineEmits<{
  close: []
}>()

const i18n = useI18n()
const { t } = i18n
const appStore = getActivePinia() ? useAppStore() : null

const generating = ref(false)
const copyingImage = ref(false)
const errorMessage = ref('')
const poster = ref<GeneratedAffiliatePoster | null>(null)
let renderSeq = 0

const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || 'zh')
  }
  return 'zh'
})

const normalizedCode = computed(() => props.affCode.trim())
const resolvedLink = computed(() =>
  props.inviteLink?.trim() || (normalizedCode.value ? buildInviteLink(normalizedCode.value) : ''),
)

function resolveBrandName(): string {
  const translated = t('home.landing.brandName')
  if (translated && translated !== 'home.landing.brandName') return translated
  return isZhLocale(localeCode.value) ? '石头中转站' : 'yf-mail'
}

async function renderPoster(): Promise<void> {
  if (!normalizedCode.value) return
  const seq = ++renderSeq
  generating.value = true
  errorMessage.value = ''
  try {
    const result = await generateAffiliatePoster({
      affCode: normalizedCode.value,
      inviteLink: resolvedLink.value,
      brandName: resolveBrandName(),
      locale: localeCode.value,
    })
    if (seq !== renderSeq) return
    poster.value = result
  } catch {
    if (seq !== renderSeq) return
    poster.value = null
    errorMessage.value = t('affiliate.poster.generateFailed')
  } finally {
    if (seq === renderSeq) {
      generating.value = false
    }
  }
}

function downloadPoster(): void {
  if (!poster.value) return
  const a = document.createElement('a')
  a.href = poster.value.dataUrl
  a.download = `invite-poster-${normalizedCode.value || 'share'}.png`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  appStore?.showSuccess(t('affiliate.poster.downloaded'))
}

async function copyPosterImage(): Promise<void> {
  if (!poster.value?.blob || copyingImage.value) {
    appStore?.showInfo(t('affiliate.poster.copyImageUnsupported'))
    return
  }
  const canWriteImage =
    typeof navigator !== 'undefined' &&
    Boolean(navigator.clipboard?.write) &&
    typeof ClipboardItem !== 'undefined'

  if (!canWriteImage) {
    appStore?.showInfo(t('affiliate.poster.copyImageUnsupported'))
    return
  }

  copyingImage.value = true
  try {
    const item = new ClipboardItem({ 'image/png': poster.value.blob })
    await navigator.clipboard.write([item])
    appStore?.showSuccess(t('affiliate.poster.imageCopied'))
  } catch {
    appStore?.showInfo(t('affiliate.poster.copyImageUnsupported'))
  } finally {
    copyingImage.value = false
  }
}

async function copyShareText(): Promise<void> {
  if (!normalizedCode.value) return
  const text = buildShareCopy(normalizedCode.value, undefined, resolveBrandName(), localeCode.value)
  const ok = await copyTextToClipboard(text)
  if (ok) {
    appStore?.showSuccess(t('affiliate.poster.captionCopied'))
  } else {
    appStore?.showError(t('common.copyFailed'))
  }
}

watch(
  () => [props.show, normalizedCode.value, resolvedLink.value, localeCode.value] as const,
  ([visible, code]) => {
    if (visible && code) {
      void renderPoster()
    }
  },
  { immediate: true },
)
</script>
