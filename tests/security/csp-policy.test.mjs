import assert from 'node:assert/strict'
import fs from 'node:fs/promises'
import test from 'node:test'

const policies = [
  {
    file: new URL('../../apps/web/index.html', import.meta.url),
    imageOrigins: [
      "'self'",
      'data:',
      'blob:',
      'https://storage.googleapis.com',
      'https://images.unsplash.com',
      'https://*.cashfree.com',
      'https://lh3.googleusercontent.com',
    ],
  },
  {
    file: new URL('../../apps/vendor-portal/index.html', import.meta.url),
    imageOrigins: [
      "'self'",
      'data:',
      'blob:',
      'https://storage.googleapis.com',
      'https://images.unsplash.com',
      'https://lh3.googleusercontent.com',
    ],
  },
  {
    file: new URL('../../apps/delivery-portal/index.html', import.meta.url),
    imageOrigins: [
      "'self'",
      'data:',
      'blob:',
      'https://storage.googleapis.com',
      'https://lh3.googleusercontent.com',
    ],
  },
]

for (const policy of policies) {
  test(`${policy.file.pathname} has an allowlisted image CSP`, async () => {
    const html = await fs.readFile(policy.file, 'utf8')
    const content = html.match(/http-equiv="Content-Security-Policy"\s+content="([^"]+)"/)?.[1]
    assert.ok(content, 'CSP meta element is required')
    const imageDirective = content
      .split(';')
      .map((directive) => directive.trim())
      .find((directive) => directive.startsWith('img-src '))
    assert.ok(imageDirective, 'img-src directive is required')
    const sources = imageDirective.split(/\s+/).slice(1)
    assert.deepEqual(sources, policy.imageOrigins)
    assert.ok(!sources.includes('https:'), 'scheme-wide image loading permits attacker-controlled hosts')
  })
}
