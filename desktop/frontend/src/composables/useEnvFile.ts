import { ref } from 'vue'
import { GetEnvFile, SaveEnvFile, DetectEditors, OpenEnvFileInEditor } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

type EnvFileState = {
  path: string
  content: string
  exists: boolean
}

export type EditorInfo = {
  id: string
  name: string
  path: string
}

const envFile = ref<EnvFileState>({ path: '', content: '', exists: false })
const envContent = ref('')
const envLoading = ref(false)
const envSaving = ref(false)
const envMessage = ref('')
const envError = ref('')
const editors = ref<EditorInfo[]>([])
const editorsLoading = ref(false)
const openingEditor = ref(false)

const loadEnvFile = async () => {
  envLoading.value = true
  envError.value = ''
  envMessage.value = ''
  try {
    const data = await GetEnvFile() as EnvFileState
    envFile.value = data
    envContent.value = data.content || ''
  } catch (error) {
    envError.value = error instanceof Error ? error.message : String(error)
  } finally {
    envLoading.value = false
  }
}

const saveEnvFile = async (content?: string) => {
  envSaving.value = true
  envMessage.value = ''
  envError.value = ''
  try {
    const nextContent = content ?? envContent.value
    await SaveEnvFile(nextContent)
    envContent.value = nextContent
    await loadEnvFile()
    envMessage.value = '.env 已保存，重启服务后生效'
  } catch (error) {
    envError.value = error instanceof Error ? error.message : String(error)
  } finally {
    envSaving.value = false
  }
}

const loadEditors = async () => {
  editorsLoading.value = true
  try {
    const list = await DetectEditors() as EditorInfo[]
    editors.value = list ?? []
  } catch {
    editors.value = []
  } finally {
    editorsLoading.value = false
  }
}

const openInEditor = async (editorPath: string) => {
  openingEditor.value = true
  envError.value = ''
  envMessage.value = ''
  try {
    await OpenEnvFileInEditor(editorPath)
    envMessage.value = '已在编辑器中打开 .env 文件'
  } catch (error) {
    envError.value = error instanceof Error ? error.message : String(error)
  } finally {
    openingEditor.value = false
  }
}

export function useEnvFile() {
  return {
    envFile,
    envContent,
    envLoading,
    envSaving,
    envMessage,
    envError,
    editors,
    editorsLoading,
    openingEditor,
    loadEnvFile,
    saveEnvFile,
    loadEditors,
    openInEditor,
  }
}
