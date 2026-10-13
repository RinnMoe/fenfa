export interface UploadResponse {
  ok: boolean
  data?: any
  error?: { message: string }
}

interface UploadRequest {
  url: string
  token: string
  formData: FormData
  withCredentials?: boolean
  onProgress?: (percent: number) => void
  onAuthError?: () => void
}

export function uploadWithProgress(options: UploadRequest): Promise<UploadResponse> {
  return new Promise((resolve) => {
    const xhr = new XMLHttpRequest()

    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        options.onProgress?.(Math.round((event.loaded / event.total) * 100))
      }
    })

    xhr.addEventListener('load', () => {
      if (xhr.status === 401 || xhr.status === 403) {
        options.onAuthError?.()
        resolve({ ok: false, error: { message: 'AUTH_ERROR' } })
        return
      }

      if (xhr.status === 200 || xhr.status === 201) {
        try {
          resolve(JSON.parse(xhr.responseText))
        } catch {
          resolve({ ok: true, data: null })
        }
        return
      }

      let message = `HTTP ${xhr.status}`
      try {
        const body = JSON.parse(xhr.responseText)
        if (body?.error?.message) message = body.error.message
      } catch {}
      resolve({ ok: false, error: { message } })
    })

    xhr.addEventListener('error', () => {
      resolve({ ok: false, error: { message: 'Network error' } })
    })

		xhr.open('POST', options.url)
		xhr.withCredentials = options.withCredentials ?? true
    if (options.token) xhr.setRequestHeader('X-Auth-Token', options.token)
    xhr.send(options.formData)
  })
}
