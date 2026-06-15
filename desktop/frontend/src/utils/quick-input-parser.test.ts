import { describe, expect, it } from 'vitest'
import { parseQuickInput } from './quick-input-parser'

describe('quick input service type detection', () => {
  it('detects Claude Messages endpoints and strips the protocol path', () => {
    const result = parseQuickInput('https://api.example.com/v1/messages sk-key1234567890')

    expect(result.detectedServiceType).toBe('claude')
    expect(result.detectedBaseUrl).toBe('https://api.example.com')
  })

  it('detects OpenAI Chat endpoints and strips the protocol path', () => {
    const result = parseQuickInput('https://api.example.com/v1/chat/completions sk-key1234567890')

    expect(result.detectedServiceType).toBe('openai')
    expect(result.detectedBaseUrl).toBe('https://api.example.com')
  })

  it('detects Responses endpoints and strips the protocol path', () => {
    const result = parseQuickInput('https://api.example.com/v1/responses sk-key1234567890')

    expect(result.detectedServiceType).toBe('responses')
    expect(result.detectedBaseUrl).toBe('https://api.example.com')
  })

  it('detects Gemini generateContent endpoints and strips the protocol path', () => {
    const result = parseQuickInput('https://generativelanguage.googleapis.com/v1beta/models/gemini-3-pro:generateContent AIza1234567890abcdefghijklmnopqrstuv')

    expect(result.detectedServiceType).toBe('gemini')
    expect(result.detectedBaseUrl).toBe('https://generativelanguage.googleapis.com')
  })

  it('detects common Anthropic and Claude URL hints without stripping non-protocol paths', () => {
    const result = parseQuickInput('https://relay.example.com/anthropic sk-key1234567890')

    expect(result.detectedServiceType).toBe('claude')
    expect(result.detectedBaseUrl).toBe('https://relay.example.com/anthropic')
  })
})
