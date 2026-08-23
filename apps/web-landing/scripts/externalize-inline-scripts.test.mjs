import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { externalizeInlineScripts } from './externalize-inline-scripts.mjs'

test('externalizes executable inline scripts and preserves inert JSON-LD', async () => {
  const outputDir = await fs.mkdtemp(path.join(os.tmpdir(), 'landing-csp-'))
  await fs.mkdir(path.join(outputDir, 'nested'))
  const executable = 'self.__next_f.push([1,"payload"]);'
  const html = `<!doctype html><html><head>
    <script>${executable}</script>
    <script type="application/ld+json">{"@type":"Organization"}</script>
    <script src="/_next/static/app.js"></script>
  </head></html>`
  await fs.writeFile(path.join(outputDir, 'index.html'), html)
  await fs.writeFile(path.join(outputDir, 'nested', 'index.html'), html)

  const result = await externalizeInlineScripts(outputDir)

  assert.equal(result.htmlFiles, 2)
  assert.equal(result.externalizedScripts, 2)
  assert.equal(result.uniqueAssets, 1, 'identical Next.js bootstraps should share one asset')
  const transformed = await fs.readFile(path.join(outputDir, 'index.html'), 'utf8')
  assert.doesNotMatch(transformed, new RegExp(`<script>${escapeRegex(executable)}</script>`))
  assert.match(transformed, /<script src="\/_next\/static\/inline\/[a-f0-9]{64}\.js"><\/script>/)
  assert.match(transformed, /<script type="application\/ld\+json">{"@type":"Organization"}<\/script>/)
  assert.match(transformed, /<script src="\/_next\/static\/app\.js"><\/script>/)

  const assets = await fs.readdir(path.join(outputDir, '_next', 'static', 'inline'))
  assert.equal(assets.length, 1)
  assert.equal(await fs.readFile(path.join(outputDir, '_next', 'static', 'inline', assets[0]), 'utf8'), executable)
})

test('externalizes inline module scripts and ignores empty script elements', async () => {
  const outputDir = await fs.mkdtemp(path.join(os.tmpdir(), 'landing-csp-'))
  await fs.writeFile(
    path.join(outputDir, 'index.html'),
    '<script type="module">globalThis.ready = true</script><script>   </script>'
  )

  await externalizeInlineScripts(outputDir)

  const transformed = await fs.readFile(path.join(outputDir, 'index.html'), 'utf8')
  assert.match(transformed, /<script type="module" src="\/_next\/static\/inline\/[a-f0-9]{64}\.js"><\/script>/)
  assert.match(transformed, /<script>   <\/script>/)
})

test('nginx CSP never permits executable unsafe-inline scripts', async () => {
	const nginx = await fs.readFile(new URL('../nginx.conf', import.meta.url), 'utf8')
	const policies = [...nginx.matchAll(/Content-Security-Policy\s+"([^"]+)"/g)].map(
		(match) => match[1]
	)
	assert.ok(policies.length >= 3)
	for (const policy of policies) {
		const scriptSource = policy
			.split(';')
			.map((directive) => directive.trim())
			.find((directive) => directive.startsWith('script-src '))
		assert.ok(scriptSource)
		assert.doesNotMatch(scriptSource, /'unsafe-inline'/)
	}
})

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
