import assert from 'node:assert/strict'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { createRequire } from 'node:module'
import test from 'node:test'

const require = createRequire(import.meta.url)

test('image-size rejects a zero-length ICNS entry without blocking the event loop', () => {
  const input = Buffer.alloc(24)
  input.write('icns', 0, 'ascii')
  input.writeUInt32BE(input.length, 4)
  input.write('ic07', 8, 'ascii')
  input.writeUInt32BE(8, 12)
  input.write('ic08', 16, 'ascii')
  input.writeUInt32BE(0, 20)

  assertParserTerminates(input)
})

test('image-size rejects a zero-length JXL partial-stream box without blocking', () => {
  const input = Buffer.alloc(40)
  input.writeUInt32BE(12, 0)
  input.write('JXL ', 4, 'ascii')
  input.set([0x0d, 0x0a, 0x87, 0x0a], 8)
  input.writeUInt32BE(20, 12)
  input.write('ftyp', 16, 'ascii')
  input.write('jxl ', 20, 'ascii')
  input.writeUInt32BE(0, 24)
  input.write('jxl ', 28, 'ascii')
  input.writeUInt32BE(0, 32)
  input.write('jxlp', 36, 'ascii')

  assertParserTerminates(input)
})

test('image-size box parser rejects a zero-length recognized HEIF box', () => {
  const imageSizeEntry = require.resolve('image-size')
  const utils = require(path.join(path.dirname(imageSizeEntry), 'types', 'utils.js'))
  const input = Buffer.alloc(8)
  input.writeUInt32BE(0, 0)
  input.write('meta', 4, 'ascii')

  assert.equal(utils.findBox(input, 'meta', 0), undefined)
})

function assertParserTerminates(input) {
  const child = spawnSync(
    process.execPath,
    [
      '-e',
      `const { imageSize } = require('image-size')
       try { imageSize(Buffer.from(process.argv[1], 'base64')) } catch {}`,
      input.toString('base64'),
    ],
    { timeout: 750, encoding: 'utf8' }
  )

  assert.equal(child.error, undefined, child.error?.message)
  assert.equal(child.status, 0, child.stderr)
  assert.notEqual(child.error?.code, 'ETIMEDOUT', 'malicious image blocked the Node.js event loop')
}
