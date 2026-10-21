import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { createServer } from 'vite'

const state = { calls: [], hit: false, fail: false }
globalThis.__imageTest = state
const server = await createServer({
  configFile: false,
  optimizeDeps: { noDiscovery: true, include: [] },
  root: process.cwd(),
  resolve: { alias: { '@': `${process.cwd()}/src` } },
  plugins: [{
    name: 'image-test-api',
    load(id) {
      if (id.replaceAll('\\', '/').endsWith('/src/request/index.ts'))
        return `export const api = { post: async config => {
          const state = globalThis.__imageTest;
          state.calls.push(config);
          config.signal?.throwIfAborted();
          if (state.fail) throw new Error('图片服务暂时不可用');
          return {data: {exists: state.hit, url: state.hit || config.url.includes('upload') ? 'https://images.example/test.png' : '', reused: state.hit}};
        }};`
    },
  }],
  server: { middlewareMode: true, watch: null },
})
try {
  const { imageMD5, uploadDocumentImage } = await server.ssrLoadModule('/src/api/images.ts')
  for (const value of ['', 'abc', '中文图片', 'x'.repeat(3 * 1024 * 1024)])
    assert.equal(await imageMD5(new Blob([value])), createHash('md5').update(value).digest('hex'))
  const file = new File(['image bytes'], 'picture.png', { type: 'image/png' })
  const options = { signal: new AbortController().signal }
  state.hit = true
  assert.equal((await uploadDocumentImage(file, options)).reused, true)
  assert.equal(state.calls.length, 1)
  assert.equal(state.calls[0].url, '/images/check')
  state.calls.length = 0
  state.hit = false
  await uploadDocumentImage(file, options)
  assert.deepEqual(state.calls.map(call => call.url), ['/images/check', '/images/upload'])
  assert.equal(state.calls[1].data.get('md5'), createHash('md5').update('image bytes').digest('hex'))
  assert.equal(state.calls[1].data.get('file').name, 'picture.png')
  assert.equal(state.calls[1].headers, undefined, 'browser owns the multipart boundary')
  state.calls.length = 0
  await assert.rejects(uploadDocumentImage(new File(['svg'], 'test.svg', { type: 'image/svg+xml' }), options), /仅支持/)
  await assert.rejects(uploadDocumentImage(new File([], 'empty.png', { type: 'image/png' }), options), /不能为空/)
  await assert.rejects(uploadDocumentImage(new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'big.png', { type: 'image/png' }), options), /不能超过/)
  const abort = new AbortController()
  abort.abort()
  await assert.rejects(uploadDocumentImage(file, { signal: abort.signal }), error => error.name === 'AbortError')
  assert.equal(state.calls.length, 0)
  state.fail = true
  await assert.rejects(uploadDocumentImage(file, options), /暂时不可用/)
  console.log('PASS MD5 vectors/chunking, cache hit without upload, cache miss, multipart, validation, cancellation and failure')
} finally {
  await server.close()
  delete globalThis.__imageTest
}
