import { describe, expect, it } from 'vitest'
import { buildConversationTurnPreview } from './conversationPreview'

describe('buildConversationTurnPreview', () => {
  it('limits a single turn to five rendered lines', () => {
    const text = 'aaaaa bbbbb ccccc ddddd eeeee fffff'
    const preview = buildConversationTurnPreview(text, {
      width: 5,
      font: '12px sans-serif',
      maxLines: 5,
      measureText: (value) => value.length,
    })

    const lines = preview.split('\n')
    expect(lines).toHaveLength(5)
    expect(lines[4]).toBe('eeee…')
  })
})
