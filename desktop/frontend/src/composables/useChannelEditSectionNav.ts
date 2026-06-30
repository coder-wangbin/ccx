import { computed, ref, type Ref } from 'vue'

type Translator = (key: string, params?: Record<string, string>) => string

export function useChannelEditSectionNav(t: Translator, dialogRef: Ref<HTMLElement | null>) {
  const activeSection = ref('basic')
  const sectionRefs = ref<Record<string, HTMLElement | null>>({})
  let scrollRoot: Element | null = null
  let scrollHandler: (() => void) | null = null

  const sections = computed(() => [
    { id: 'basic', label: t('channelEditor.nav.basic') },
    { id: 'auth', label: t('channelEditor.nav.auth') },
    { id: 'redirect', label: t('channelEditor.nav.redirect') },
    { id: 'advanced', label: t('channelEditor.nav.advanced') },
    { id: 'custom', label: t('channelEditor.nav.custom') },
  ])

  function scrollToSection(id: string) {
    activeSection.value = id
    const el = sectionRefs.value[id]
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  function setSectionRef(id: string, el: any) {
    sectionRefs.value[id] = el as HTMLElement | null
  }

  function updateActiveSectionFromScroll() {
    if (!scrollRoot) return
    const rootTop = scrollRoot.getBoundingClientRect().top
    let current = sections.value[0]?.id || 'basic'

    for (const s of sections.value) {
      const el = sectionRefs.value[s.id]
      if (!el) continue
      const top = el.getBoundingClientRect().top - rootTop
      if (top <= 60) {
        current = s.id
      } else {
        break
      }
    }

    if (activeSection.value !== current) {
      activeSection.value = current
    }
  }

  function bindScrollRoot() {
    let viewport = dialogRef.value?.querySelector('[data-slot="scroll-area-viewport"]') as Element | null
    if (!viewport) {
      console.warn('[ChannelEditDialog] dialogRef 查询失败，尝试全局查询')
      const all = document.querySelectorAll('[data-slot="scroll-area-viewport"]')
      viewport = all.length > 0 ? all[all.length - 1] : null
    }

    if (!viewport) {
      console.error('[ChannelEditDialog] 未找到滚动容器')
      return
    }

    scrollRoot = viewport
    console.log('[ChannelEditDialog] 滚动容器已绑定', scrollRoot)
    scrollHandler = () => updateActiveSectionFromScroll()
    scrollRoot.addEventListener('scroll', scrollHandler, { passive: true })
    updateActiveSectionFromScroll()
  }

  function unbindScrollRoot() {
    if (scrollRoot && scrollHandler) {
      scrollRoot.removeEventListener('scroll', scrollHandler)
    }
    scrollRoot = null
    scrollHandler = null
  }

  return {
    activeSection,
    sections,
    scrollToSection,
    setSectionRef,
    updateActiveSectionFromScroll,
    bindScrollRoot,
    unbindScrollRoot,
  }
}
