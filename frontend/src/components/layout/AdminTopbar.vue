<script setup lang="ts">
import { Fold, Moon, Sunny } from '@element-plus/icons-vue'
import { computed, onMounted, ref, watch } from 'vue'

defineProps<{
  pageDescription?: string
  pageTitle: string
}>()

const emit = defineEmits<{
  (event: 'toggle-sidebar'): void
}>()

// 主题配置保存在浏览器本地。
// 下次刷新页面时，会沿用上次选中的主题色和亮暗模式。
const THEME_STORAGE_KEY = 'netweaver-dashboard-theme'

const themeHue = ref(174)
const isDimMode = ref(false)

const themeModeText = computed(() => (isDimMode.value ? '暗色' : '亮色'))
const themeButtonLabel = computed(() => `切换到${isDimMode.value ? '亮色' : '暗色'}模式`)

const applyTheme = () => {
  document.documentElement.style.setProperty('--theme-hue', String(themeHue.value))
  document.documentElement.dataset.themeMode = isDimMode.value ? 'dim' : 'light'
  window.dispatchEvent(new CustomEvent('netweaver-theme-change'))
}

const saveTheme = () => {
  window.localStorage.setItem(
    THEME_STORAGE_KEY,
    JSON.stringify({
      hue: themeHue.value,
      dim: isDimMode.value
    })
  )
}

const loadTheme = () => {
  const rawValue = window.localStorage.getItem(THEME_STORAGE_KEY)

  if (!rawValue) {
    return
  }

  try {
    const savedTheme = JSON.parse(rawValue) as { hue?: number; dim?: boolean }

    if (typeof savedTheme.hue === 'number' && savedTheme.hue >= 0 && savedTheme.hue <= 360) {
      themeHue.value = savedTheme.hue
    }

    if (typeof savedTheme.dim === 'boolean') {
      isDimMode.value = savedTheme.dim
    }
  } catch {
    // 本地存储被手动改坏时，直接忽略并使用默认主题。
  }
}

const toggleDimMode = () => {
  isDimMode.value = !isDimMode.value
}

watch([themeHue, isDimMode], () => {
  applyTheme()
  saveTheme()
})

onMounted(() => {
  loadTheme()
  applyTheme()
})
</script>

<template>
  <header class="panel-surface sticky top-4 z-20 px-4 py-4 sm:px-6">
    <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
      <div class="flex min-w-0 items-center gap-3">
        <el-button class="lg:hidden" circle plain @click="emit('toggle-sidebar')">
          <el-icon><Fold /></el-icon>
        </el-button>

        <div class="min-w-0">
          <h1 class="truncate text-2xl font-semibold tracking-[0.04em] text-slate-900">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="mt-1 truncate text-sm text-slate-500">
            {{ pageDescription }}
          </p>
        </div>
      </div>

      <div class="flex shrink-0 flex-wrap items-center gap-3">
        <section class="theme-controls" aria-label="主题外观设置">
          <div class="theme-color-control">
            <span class="theme-color-control__label">主题色</span>
            <el-slider
              v-model="themeHue"
              :max="360"
              :min="0"
              :show-tooltip="false"
              :step="0.1"
              class="theme-hue-slider"
              aria-label="连续调整主题色"
            />
          </div>

          <button
            class="theme-light-toggle"
            :class="{ 'is-dim': isDimMode }"
            type="button"
            :aria-label="themeButtonLabel"
            :title="themeButtonLabel"
            @click="toggleDimMode"
          >
            <span class="theme-light-toggle__glow" />
            <el-icon class="relative z-10 text-base">
              <Moon v-if="isDimMode" />
              <Sunny v-else />
            </el-icon>
            <span class="relative z-10 text-xs font-semibold">{{ themeModeText }}</span>
          </button>
        </section>

        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<style scoped>
.theme-controls {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  border: 1px solid var(--app-panel-border);
  border-radius: 999px;
  background: var(--app-control-bg);
  padding: 0.45rem 0.55rem 0.45rem 0.9rem;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.36);
  transition:
    background-color 0.45s ease,
    border-color 0.45s ease,
    box-shadow 0.45s ease;
}

.theme-color-control {
  display: grid;
  grid-template-columns: auto minmax(7.5rem, 10rem);
  align-items: center;
  gap: 0.75rem;
}

.theme-color-control__label {
  color: var(--app-text-soft);
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.16em;
}

.theme-light-toggle {
  position: relative;
  display: inline-flex;
  min-width: 5.3rem;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  overflow: hidden;
  border: 1px solid var(--app-primary-border);
  border-radius: 999px;
  background:
    radial-gradient(circle at 35% 28%, rgba(255, 255, 255, 0.85), transparent 34%),
    linear-gradient(135deg, var(--app-primary-soft), var(--app-primary));
  color: var(--app-primary-strong);
  padding: 0.5rem 0.75rem;
  box-shadow:
    0 0 22px var(--app-primary-glow),
    inset 0 1px 0 rgba(255, 255, 255, 0.45);
  cursor: pointer;
  transition:
    transform 0.25s ease,
    color 0.45s ease,
    background 0.45s ease,
    border-color 0.45s ease,
    box-shadow 0.45s ease;
}

.theme-light-toggle:hover {
  transform: translateY(-1px);
  box-shadow:
    0 0 34px var(--app-primary-glow-strong),
    inset 0 1px 0 rgba(255, 255, 255, 0.52);
}

.theme-light-toggle.is-dim {
  color: var(--app-text);
  background:
    radial-gradient(circle at 65% 35%, var(--app-primary), transparent 27%),
    linear-gradient(135deg, rgba(15, 23, 42, 0.92), var(--app-panel-bg-solid));
}

.theme-light-toggle__glow {
  position: absolute;
  inset: -45%;
  background: conic-gradient(from 120deg, transparent, var(--app-primary-glow-strong), transparent);
  opacity: 0.74;
  animation: theme-light-spin 7s linear infinite;
}

:deep(.theme-hue-slider.el-slider) {
  --el-slider-button-size: 16px;
  width: 100%;
  min-width: 7.5rem;
}

:deep(.theme-hue-slider .el-slider__runway) {
  height: 0.5rem;
  border-radius: 999px;
  background: linear-gradient(90deg, #ef4444, #f59e0b, #eab308, #22c55e, #06b6d4, #3b82f6, #8b5cf6, #ec4899, #ef4444);
  box-shadow: inset 0 0 0 1px rgba(15, 23, 42, 0.14);
}

:deep(.theme-hue-slider .el-slider__bar) {
  background: transparent;
}

:deep(.theme-hue-slider .el-slider__button) {
  border: 3px solid #ffffff;
  background: var(--app-primary);
  box-shadow:
    0 0 0 1px var(--app-primary-border),
    0 0 18px var(--app-primary-glow-strong);
}

@keyframes theme-light-spin {
  to {
    transform: rotate(1turn);
  }
}

@media (max-width: 640px) {
  .theme-controls {
    width: 100%;
    justify-content: space-between;
    border-radius: 1.35rem;
  }

  .theme-color-control {
    flex: 1;
    grid-template-columns: auto minmax(6rem, 1fr);
  }
}
</style>
