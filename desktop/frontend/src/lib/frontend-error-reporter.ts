import { ReportFrontendError } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

type ReportSource = 'window-error' | 'unhandled-rejection' | 'console-error'

type SerializedError = {
  message: string
  stack: string
}

let installed = false

export function installFrontendErrorReporter() {
  if (installed || typeof window === 'undefined') {
    return
  }
  installed = true

  window.addEventListener('error', (event) => {
    const serialized = serializeErrorLike(event.error, event.message)
    const location = [event.filename, event.lineno, event.colno].filter(Boolean).join(':')
    reportFrontendError('window-error', {
      message: serialized.message,
      stack: [serialized.stack, location].filter(Boolean).join('\n'),
    })
  })

  window.addEventListener('unhandledrejection', (event) => {
    reportFrontendError('unhandled-rejection', serializeErrorLike(event.reason))
  })

  const originalConsoleError = console.error.bind(console)
  console.error = (...args: unknown[]) => {
    originalConsoleError(...args)
    reportFrontendError('console-error', serializeConsoleArgs(args))
  }
}

function reportFrontendError(source: ReportSource, error: SerializedError) {
  if (!isWailsWebView()) {
    return
  }

  const message = redactSecrets(error.message)
  const stack = redactSecrets(error.stack)
  if (!message && !stack) {
    return
  }

  void ReportFrontendError({
    source,
    message,
    stack,
    url: window.location.href,
    userAgent: navigator.userAgent,
  }).catch(() => undefined)
}

function isWailsWebView() {
  const wails = (window as unknown as { _wails?: { environment?: { OS?: string }; invoke?: unknown } })._wails
  return Boolean(wails?.environment?.OS && typeof wails.invoke === 'function')
}

function serializeConsoleArgs(args: unknown[]): SerializedError {
  return {
    message: args.map((arg) => serializeValue(arg)).join(' '),
    stack: '',
  }
}

function serializeErrorLike(value: unknown, fallback = ''): SerializedError {
  if (value instanceof Error) {
    return {
      message: value.message || fallback,
      stack: value.stack || '',
    }
  }
  return {
    message: serializeValue(value || fallback),
    stack: '',
  }
}

function serializeValue(value: unknown): string {
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' || typeof value === 'boolean' || value == null) {
    return String(value)
  }
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function redactSecrets(value: string): string {
  return value
    .replace(/(authorization\s*[:=]\s*bearer\s+)[^\s"',}]+/gi, '$1[redacted]')
    .replace(/((?:api[_-]?key|x-api-key|proxy_access_key|admin_access_key)\s*[:=]\s*)["']?[^"',}\s]+/gi, '$1[redacted]')
}
