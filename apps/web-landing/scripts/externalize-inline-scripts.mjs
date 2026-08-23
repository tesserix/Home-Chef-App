import { createHash } from 'node:crypto'
import fs from 'node:fs/promises'
import path from 'node:path'
import { pathToFileURL } from 'node:url'

const scriptPattern = /<script\b([^>]*)>([\s\S]*?)<\/script>/gi
const executableTypes = new Set([
  '',
  'module',
  'text/javascript',
  'application/javascript',
  'text/ecmascript',
  'application/ecmascript',
])

export async function externalizeInlineScripts(outputDir) {
  const htmlPaths = (await walk(outputDir)).filter((file) => file.endsWith('.html'))
  const assets = new Map()
  let externalizedScripts = 0

  for (const htmlPath of htmlPaths) {
    const html = await fs.readFile(htmlPath, 'utf8')
    const transformed = html.replace(scriptPattern, (element, attributes, body) => {
      if (/\bsrc\s*=/i.test(attributes) || !body.trim() || !isExecutable(attributes)) {
        return element
      }
      const digest = createHash('sha256').update(body).digest('hex')
      assets.set(digest, body)
      externalizedScripts++
      return `<script${attributes} src="/_next/static/inline/${digest}.js"></script>`
    })
    assertNoExecutableInlineScripts(transformed, htmlPath)
    if (transformed !== html) {
      await fs.writeFile(htmlPath, transformed)
    }
  }

  const assetDir = path.join(outputDir, '_next', 'static', 'inline')
  await fs.mkdir(assetDir, { recursive: true })
  await Promise.all(
    [...assets].map(([digest, body]) => fs.writeFile(path.join(assetDir, `${digest}.js`), body))
  )

  return {
    htmlFiles: htmlPaths.length,
    externalizedScripts,
    uniqueAssets: assets.size,
  }
}

function isExecutable(attributes) {
  const match = attributes.match(/\btype\s*=\s*(["'])(.*?)\1/i)
  const type = match ? match[2].trim().toLowerCase() : ''
  return executableTypes.has(type)
}

function assertNoExecutableInlineScripts(html, htmlPath) {
  for (const match of html.matchAll(scriptPattern)) {
    const [, attributes, body] = match
    if (!/\bsrc\s*=/i.test(attributes) && body.trim() && isExecutable(attributes)) {
      throw new Error(`executable inline script remains in ${htmlPath}`)
    }
  }
}

async function walk(directory) {
  const entries = await fs.readdir(directory, { withFileTypes: true })
  const files = []
  for (const entry of entries) {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await walk(entryPath)))
    } else if (entry.isFile()) {
      files.push(entryPath)
    }
  }
  return files
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  const outputDir = path.resolve(process.argv[2] || 'out')
  const result = await externalizeInlineScripts(outputDir)
  console.log(
    `CSP externalization complete: html_files=${result.htmlFiles} ` +
      `scripts=${result.externalizedScripts} unique_assets=${result.uniqueAssets}`
  )
}
