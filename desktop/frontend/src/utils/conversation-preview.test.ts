import { describe, expect, it } from 'vitest'
import { buildConversationTurnMiddlePreview, buildConversationTurnPreview } from './conversation-preview'

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

describe('buildConversationTurnMiddlePreview', () => {
  it('keeps the first and last two rendered lines', () => {
    const preview = buildConversationTurnMiddlePreview(
      'one two three four five six seven eight nine ten',
      {
        width: 5,
        font: '12px sans-serif',
        edgeLines: 2,
        measureText: (value) => value.length,
      },
    )

    expect(preview).toEqual({
      head: 'one\ntwo',
      tail: 'nine\nten',
      truncated: true,
    })
  })

  it('does not truncate short text', () => {
    const preview = buildConversationTurnMiddlePreview('one two', {
      width: 5,
      font: '12px sans-serif',
      edgeLines: 2,
      measureText: (value) => value.length,
    })

    expect(preview).toEqual({
      head: 'one two',
      tail: '',
      truncated: false,
    })
  })
})
