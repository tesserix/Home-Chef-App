import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import test from 'node:test'

const require = createRequire(import.meta.url)

test('@tesserix/native patch applies to source and both published entrypoints', async () => {
  const cjsEntry = require.resolve('@tesserix/native')
  const packageRoot = path.dirname(path.dirname(cjsEntry))
  const files = [
    path.join(packageRoot, 'src', 'components', 'Button', 'Button.tsx'),
    path.join(packageRoot, 'dist', 'index.js'),
    path.join(packageRoot, 'dist', 'index.mjs'),
  ]

  for (const file of files) {
    const contents = await fs.readFile(file, 'utf8')
    const buttonStart = contents.indexOf('resolvedColorScheme')
    assert.notEqual(buttonStart, -1, `Button color logic missing from ${file}`)
    const buttonSection = contents.slice(buttonStart, buttonStart + 4000)
    assert.match(buttonSection, /#C2410C/)
    assert.match(buttonSection, /#B22B0E/)
    assert.match(buttonSection, /#1A1A18/)
  }
})
