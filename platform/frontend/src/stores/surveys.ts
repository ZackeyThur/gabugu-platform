import { defineStore } from 'pinia'
import { ref } from 'vue'
import { surveysAPI } from '@/api'
import type { SubmitSurveyResult, SurveyAnswers, UserSurvey } from '@/types/survey'

const THROTTLE_MS = 10 * 60 * 1000 // 10 minutes

export const useSurveyStore = defineStore('surveys', () => {
  const surveys = ref<UserSurvey[]>([])
  const currentSurvey = ref<UserSurvey | null>(null)
  const minimized = ref(false)
  const loading = ref(false)
  const lastFetchTime = ref(0)

  let handledIds = new Set<number>()
  let fetchGeneration = 0
  let delayTimer: ReturnType<typeof setTimeout> | null = null

  function clearDelayTimer() {
    if (delayTimer) {
      clearTimeout(delayTimer)
      delayTimer = null
    }
  }

  async function fetchActiveSurveys(force = false) {
    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return
    }

    lastFetchTime.value = now
    const generation = ++fetchGeneration

    try {
      loading.value = true
      const items = await surveysAPI.listActive()
      if (generation !== fetchGeneration) return
      surveys.value = Array.isArray(items) ? items : []
      scheduleNextSurvey()
    } catch (err) {
      if (generation !== fetchGeneration) return
      lastFetchTime.value = 0
      console.error('Failed to fetch active surveys:', err)
    } finally {
      if (generation === fetchGeneration) {
        loading.value = false
      }
    }
  }

  function scheduleNextSurvey() {
    if (currentSurvey.value) return
    const next = surveys.value.find((s) => !handledIds.has(s.id))
    if (!next) return

    clearDelayTimer()
    const delayMs = Math.max(0, (next.delay_seconds ?? 4) * 1000)
    if (delayMs === 0) {
      currentSurvey.value = next
      minimized.value = false
      return
    }
    delayTimer = setTimeout(() => {
      delayTimer = null
      if (!currentSurvey.value && !handledIds.has(next.id)) {
        currentSurvey.value = next
        minimized.value = false
      }
    }, delayMs)
  }

  async function submitCurrent(
    answers: SurveyAnswers,
    durationSeconds: number
  ): Promise<SubmitSurveyResult | null> {
    if (!currentSurvey.value) return null
    const id = currentSurvey.value.id
    const result = await surveysAPI.submit(id, {
      answers,
      duration_seconds: Math.max(0, Math.round(durationSeconds))
    })
    handledIds.add(id)
    surveys.value = surveys.value.filter((s) => s.id !== id)
    return result
  }

  async function snoozeCurrent() {
    if (!currentSurvey.value) return
    const id = currentSurvey.value.id
    handledIds.add(id)
    currentSurvey.value = null
    minimized.value = false
    surveys.value = surveys.value.filter((s) => s.id !== id)
    try {
      await surveysAPI.recordState(id, 'snooze')
    } catch (err) {
      console.error('Failed to snooze survey:', err)
    }
  }

  async function dismissCurrent() {
    if (!currentSurvey.value) return
    const id = currentSurvey.value.id
    handledIds.add(id)
    currentSurvey.value = null
    minimized.value = false
    surveys.value = surveys.value.filter((s) => s.id !== id)
    try {
      await surveysAPI.recordState(id, 'dismiss')
    } catch (err) {
      console.error('Failed to dismiss survey:', err)
    }
  }

  function closeAfterDone() {
    if (currentSurvey.value) {
      handledIds.add(currentSurvey.value.id)
      surveys.value = surveys.value.filter((s) => s.id !== currentSurvey.value?.id)
    }
    currentSurvey.value = null
    minimized.value = false
  }

  function setMinimized(val: boolean) {
    minimized.value = val
  }

  function reset() {
    fetchGeneration++
    clearDelayTimer()
    surveys.value = []
    currentSurvey.value = null
    minimized.value = false
    loading.value = false
    lastFetchTime.value = 0
    handledIds = new Set()
  }

  return {
    surveys,
    currentSurvey,
    minimized,
    loading,
    fetchActiveSurveys,
    submitCurrent,
    snoozeCurrent,
    dismissCurrent,
    closeAfterDone,
    setMinimized,
    reset
  }
})
