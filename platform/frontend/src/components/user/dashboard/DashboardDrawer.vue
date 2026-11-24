<template>
  <Teleport to="body">
    <Transition name="overview-drawer">
      <div v-if="open" class="overview-drawer-backdrop" @click.self="$emit('close')">
        <section ref="panel" class="overview-drawer-panel" role="dialog" aria-modal="true" :aria-labelledby="titleId" tabindex="-1">
          <header class="overview-drawer-heading">
            <div class="overview-drawer-title-group">
              <slot name="title-prefix" />
              <h2 :id="titleId">{{ title }}</h2>
            </div>
            <button type="button" class="overview-icon-button overview-drawer-close" :aria-label="t('common.close')" @click="$emit('close')">
              <Icon name="x" size="md" />
            </button>
          </header>
          <div class="overview-drawer-body"><slot /></div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const titleId = useId()
const panel = ref<HTMLElement | null>(null)
let previous: HTMLElement | null = null
let previousOverflow = ''
let previousInert = false
let locked = false

function release() {
  if (!locked) return
  document.body.style.overflow = previousOverflow
  const app = document.getElementById('app')
  if (app) app.inert = previousInert
  document.removeEventListener('keydown', onKey)
  previous?.focus()
  locked = false
}
function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') { event.preventDefault(); emit('close'); return }
  if (event.key !== 'Tab' || !panel.value) return
  const controls = [...panel.value.querySelectorAll<HTMLElement>('button:not([disabled]), a[href], input:not([disabled]), select, [tabindex="0"]')]
  const first = controls[0], last = controls.at(-1)
  if (!first) { event.preventDefault(); panel.value.focus(); return }
  if (event.shiftKey && (document.activeElement === first || document.activeElement === panel.value)) {
    event.preventDefault(); last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault(); first.focus()
  }
}
watch(() => props.open, async open => {
  if (!open) { release(); return }
  previous = document.activeElement as HTMLElement | null
  previousOverflow = document.body.style.overflow
  const app = document.getElementById('app')
  previousInert = app?.inert ?? false
  if (app) app.inert = true
  document.body.style.overflow = 'hidden'
  locked = true
  document.addEventListener('keydown', onKey)
  await nextTick()
  if (props.open) panel.value?.querySelector<HTMLElement>('button')?.focus()
}, { immediate: true })
onBeforeUnmount(release)
</script>
