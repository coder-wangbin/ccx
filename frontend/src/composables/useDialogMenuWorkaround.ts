import { ref } from 'vue'

export function useDialogMenuWorkaround() {
  const isAnySelectMenuOpen = ref(false)
  const suppressDialogEscapeUntil = ref(0)

  const onMenuUpdate = (open: boolean) => {
    isAnySelectMenuOpen.value = open
    if (!open) {
      suppressDialogEscapeUntil.value = Date.now() + 150
    }
    if (open) {
      setTimeout(() => window.dispatchEvent(new Event('resize')), 50)
    }
  }

  return {
    isAnySelectMenuOpen,
    suppressDialogEscapeUntil,
    onMenuUpdate,
  }
}
