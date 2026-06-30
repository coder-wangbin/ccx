import { reactive, ref } from 'vue'
import type { Channel } from '@/services/admin-api'

export interface HeaderRow {
  id: number
  key: string
  value: string
}

type ChannelCustomHeaderOptions = {
  nextRowId: () => number
}

export function useChannelCustomHeaders(options: ChannelCustomHeaderOptions) {
  const headerRows = ref<HeaderRow[]>([])
  const newHeader = reactive<HeaderRow>({ id: 0, key: '', value: '' })

  function headerRowsFromChannel(ch: Channel) {
    const headers = ch.customHeaders || {}
    return Object.entries(headers).map(([key, value]) => ({ id: options.nextRowId(), key, value }))
  }

  function addHeaderRow() {
    if (!newHeader.key.trim()) return
    headerRows.value.push({ id: options.nextRowId(), key: newHeader.key.trim(), value: newHeader.value })
    newHeader.key = ''
    newHeader.value = ''
  }

  function removeHeaderRow(id: number) {
    headerRows.value = headerRows.value.filter(row => row.id !== id)
  }

  function updateHeaderRow(id: number, field: 'key' | 'value', value: string) {
    const row = headerRows.value.find(r => r.id === id)
    if (row) row[field] = value
  }

  function getHeadersAsObject(): Record<string, string> {
    const result: Record<string, string> = {}
    for (const header of headerRows.value) {
      if (header.key.trim()) result[header.key.trim()] = header.value
    }
    return result
  }

  return {
    headerRows,
    newHeader,
    headerRowsFromChannel,
    addHeaderRow,
    removeHeaderRow,
    updateHeaderRow,
    getHeadersAsObject,
  }
}
