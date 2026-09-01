<template>
  <div class="space-y-2">
    <div v-for="(binding, index) in modelValue" :key="binding.proxy_id" class="flex items-center gap-2">
      <select
        :value="binding.proxy_id"
        class="input min-w-0 flex-1"
        @change="updateProxy(index, Number(($event.target as HTMLSelectElement).value))"
      >
        <option v-for="proxy in availableProxies(binding.proxy_id)" :key="proxy.id" :value="proxy.id">
          {{ proxy.name }} ({{ proxy.host }}:{{ proxy.port }})
        </option>
      </select>
      <input
        :value="binding.concurrency"
        type="number"
        min="1"
        class="input w-24"
        :aria-label="t('admin.accounts.proxyConcurrency')"
        @input="updateConcurrency(index, Number(($event.target as HTMLInputElement).value))"
      />
      <label class="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
        <input :checked="binding.enabled" type="checkbox" @change="toggleEnabled(index)" />
        {{ t('common.enabled') }}
      </label>
      <button type="button" class="btn btn-secondary px-2" :title="t('common.delete')" @click="remove(index)">
        <Icon name="trash" size="xs" />
      </button>
    </div>

    <div class="flex items-center gap-2">
      <select v-model="pendingProxyID" class="input min-w-0 flex-1">
        <option :value="null">{{ t('admin.accounts.selectProxy') }}</option>
        <option v-for="proxy in unselectedProxies" :key="proxy.id" :value="proxy.id">
          {{ proxy.name }} ({{ proxy.host }}:{{ proxy.port }})
        </option>
      </select>
      <input v-model.number="pendingConcurrency" type="number" min="1" class="input w-24" :aria-label="t('admin.accounts.proxyConcurrency')" />
      <button type="button" class="btn btn-secondary" :disabled="pendingProxyID == null" @click="add">
        <Icon name="plus" size="xs" />
        {{ t('common.add') }}
      </button>
    </div>
    <p class="input-hint">{{ t('admin.accounts.proxyBindingsHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AccountProxyBinding, Proxy } from '@/types'

const props = defineProps<{ modelValue: AccountProxyBinding[]; proxies: Proxy[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: AccountProxyBinding[]] }>()
const { t } = useI18n()
const pendingProxyID = ref<number | null>(null)
const pendingConcurrency = ref(5)

const selectedIDs = computed(() => new Set(props.modelValue.map((binding) => binding.proxy_id)))
const unselectedProxies = computed(() => props.proxies.filter((proxy) => !selectedIDs.value.has(proxy.id)))
const availableProxies = (currentID: number) => props.proxies.filter((proxy) => proxy.id === currentID || !selectedIDs.value.has(proxy.id))
const clone = () => props.modelValue.map((binding) => ({ ...binding }))

const add = () => {
  if (pendingProxyID.value == null) return
  const next = clone()
  next.push({ proxy_id: pendingProxyID.value, concurrency: Math.max(1, pendingConcurrency.value || 1), enabled: true, sort_order: next.length })
  emit('update:modelValue', next)
  pendingProxyID.value = null
}
const remove = (index: number) => {
  const next = clone()
  next.splice(index, 1)
  emit('update:modelValue', next.map((binding, order) => ({ ...binding, sort_order: order })))
}
const updateProxy = (index: number, proxyID: number) => {
  const next = clone()
  next[index].proxy_id = proxyID
  emit('update:modelValue', next)
}
const updateConcurrency = (index: number, value: number) => {
  const next = clone()
  next[index].concurrency = Math.max(1, value || 1)
  emit('update:modelValue', next)
}
const toggleEnabled = (index: number) => {
  const next = clone()
  next[index].enabled = !next[index].enabled
  emit('update:modelValue', next)
}
</script>
