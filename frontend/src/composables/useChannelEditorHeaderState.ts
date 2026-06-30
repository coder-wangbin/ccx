import { computed } from 'vue'

type ThemeLike = {
  global: {
    current: {
      value: {
        dark: boolean
      }
    }
  }
}

export function useChannelEditorHeaderState(theme: ThemeLike) {
  const headerClasses = computed(() => {
    const isDark = theme.global.current.value.dark
    return isDark ? 'bg-surface text-high-emphasis' : 'bg-primary text-white'
  })

  const avatarColor = computed(() => 'primary')

  const headerIconStyle = computed(() => ({
    color: 'rgb(var(--v-theme-on-primary))',
  }))

  const subtitleClasses = computed(() => {
    const isDark = theme.global.current.value.dark
    return isDark ? 'text-medium-emphasis' : 'text-white-subtitle'
  })

  return {
    headerClasses,
    avatarColor,
    headerIconStyle,
    subtitleClasses,
  }
}
