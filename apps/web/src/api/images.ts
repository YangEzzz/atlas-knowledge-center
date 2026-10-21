import SparkMD5 from 'spark-md5'
import { api } from '@/request'

export interface UploadedImage {
  exists?: boolean
  url: string
  md5: string
  reused: boolean
  size?: number
  mimeType?: string
}

export async function imageMD5(file: Blob, signal?: AbortSignal): Promise<string> {
  const hash = new SparkMD5.ArrayBuffer()
  try {
    for (let offset = 0; offset < file.size; offset += 2 * 1024 * 1024) {
      signal?.throwIfAborted()
      hash.append(await file.slice(offset, offset + 2 * 1024 * 1024).arrayBuffer())
    }
    signal?.throwIfAborted()
    return hash.end()
  } finally {
    hash.destroy()
  }
}

export async function uploadDocumentImage(file: File, options: {
  signal: AbortSignal
  onStatus?: (message: string) => void
}): Promise<UploadedImage> {
  if (!/\.(jpe?g|png|gif|webp)$/i.test(file.name) || !['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type))
    throw new Error('仅支持 JPG、PNG、GIF、WebP 图片')
  if (!file.size || file.size > 10 * 1024 * 1024)
    throw new Error('图片不能为空且不能超过 10MB')
  options.onStatus?.('正在计算图片 MD5…')
  const md5 = await imageMD5(file, options.signal)
  options.onStatus?.('正在检查已有图片…')
  const checked = await api.post<UploadedImage>({ url: '/images/check', data: { md5 }, signal: options.signal })
  if (checked.data.exists) {
    if (!checked.data.url)
      throw new Error('图片服务未返回地址，请重试')
    return { ...checked.data, reused: true }
  }
  options.onStatus?.('正在上传图片…')
  const form = new FormData()
  form.append('file', file)
  form.append('md5', md5)
  const response = await api.post<UploadedImage>({
    url: '/images/upload', data: form, signal: options.signal, timeout: 60000,
    onUploadProgress: event => {
      if (event.total)
        options.onStatus?.(event.loaded === event.total ? '上传完成，正在等待图片地址…' : `正在上传图片 ${Math.round(event.loaded / event.total * 100)}%`)
    },
  })
  if (!response.data.url)
    throw new Error('图片服务未返回地址，请重试')
  return response.data
}
