export type PreviewBlock =
  | { type: 'h1' | 'h2' | 'h3' | 'paragraph' | 'code'; text: string }
  | { type: 'list'; items: string[] }
  | { type: 'image'; alt: string; url: string }

const imagePattern = /^!\[([^\]]*)\]\(([^)]+)\)$/

export function parseMarkdown(content: string): PreviewBlock[] {
  const blocks: PreviewBlock[] = []
  const lines = content.split('\n')
  let listItems: string[] = []
  let codeLines: string[] = []
  let inCode = false

  const flushList = () => {
    if (listItems.length > 0) {
      blocks.push({ type: 'list', items: [...listItems] })
      listItems = []
    }
  }

  const flushCode = () => {
    if (codeLines.length > 0) {
      blocks.push({ type: 'code', text: codeLines.join('\n') })
      codeLines = []
    }
  }

  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      if (inCode) {
        inCode = false
        flushCode()
      } else {
        flushList()
        inCode = true
      }
      continue
    }

    if (inCode) {
      codeLines.push(line)
      continue
    }

    const trimmed = line.trim()
    if (!trimmed) {
      flushList()
      continue
    }

    const imageMatch = trimmed.match(imagePattern)
    if (imageMatch) {
      flushList()
      blocks.push({
        type: 'image',
        alt: imageMatch[1] || '文档图片',
        url: imageMatch[2] || '',
      })
    } else if (trimmed.startsWith('### ')) {
      flushList()
      blocks.push({ type: 'h3', text: trimmed.slice(4) })
    } else if (trimmed.startsWith('## ')) {
      flushList()
      blocks.push({ type: 'h2', text: trimmed.slice(3) })
    } else if (trimmed.startsWith('# ')) {
      flushList()
      blocks.push({ type: 'h1', text: trimmed.slice(2) })
    } else if (/^[-*]\s+/.test(trimmed)) {
      listItems.push(trimmed.replace(/^[-*]\s+/, ''))
    } else {
      flushList()
      blocks.push({ type: 'paragraph', text: trimmed })
    }
  }

  flushList()
  if (inCode) {
    flushCode()
  }
  return blocks
}

export type TocItem = { id: string; text: string; level: number }

const htmlTagPattern = /<\/?(h[1-6]|p|div|ul|ol|li|table|thead|tbody|tr|td|th|blockquote|pre|code|img|strong|em|a|br)(\s[^>]*)?>/i
const htmlHeadingPattern = /<h([1-3])([^>]*)>([\s\S]*?)<\/h\1>/gi

export function isHtmlContent(content: string) {
  return htmlTagPattern.test(content)
}

export function stripHtml(content: string) {
  return content
    .replace(/<style[\s\S]*?<\/style>/gi, '')
    .replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s+/g, ' ')
    .trim()
}

export function markdownToHtml(content: string) {
  return parseMarkdown(content)
    .map((block) => {
      if (block.type === 'h1') return `<h1>${escapeHtml(block.text)}</h1>`
      if (block.type === 'h2') return `<h2>${escapeHtml(block.text)}</h2>`
      if (block.type === 'h3') return `<h3>${escapeHtml(block.text)}</h3>`
      if (block.type === 'list') {
        return `<ul>${block.items.map((item) => `<li>${escapeHtml(item)}</li>`).join('')}</ul>`
      }
      if (block.type === 'code') return `<pre><code>${escapeHtml(block.text)}</code></pre>`
      if (block.type === 'image') {
        return `<figure><img src="${escapeHtml(block.url)}" alt="${escapeHtml(block.alt)}" /><figcaption>${escapeHtml(block.alt)}</figcaption></figure>`
      }
      return `<p>${escapeHtml(block.text)}</p>`
    })
    .join('')
}

export function extractMarkdownToc(blocks: PreviewBlock[]) {
  return blocks
    .map((block, index) => {
      if (block.type === 'h1') return { id: headingId(index), text: block.text, level: 1 }
      if (block.type === 'h2') return { id: headingId(index), text: block.text, level: 2 }
      if (block.type === 'h3') return { id: headingId(index), text: block.text, level: 3 }
      return null
    })
    .filter((item): item is TocItem => Boolean(item))
}

export function extractHtmlToc(content: string) {
  const items: TocItem[] = []
  for (const match of content.matchAll(htmlHeadingPattern)) {
    items.push({
      id: headingId(items.length),
      text: stripHtml(match[3] || ''),
      level: Number(match[1]),
    })
  }
  return items
}

export function addHeadingIdsToHtml(content: string) {
  let index = 0
  return content.replace(htmlHeadingPattern, (full, level: string, attrs: string, inner: string) => {
    if (/\sid=/.test(attrs)) {
      index += 1
      return full
    }
    const id = headingId(index)
    index += 1
    return `<h${level}${attrs} id="${id}">${inner}</h${level}>`
  })
}

function headingId(index: number) {
  return `doc-heading-${index}`
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
