#!/usr/bin/env node
/**
 * Generate patterns_node.json from the TypeScript minimatch reference.
 * Run from minimatch package root (or any cwd with relative paths adjusted):
 *
 *   node ../minimatch-go/testdata/generate_patterns.mjs
 *
 * Requires minimatch/dist built (npm test prepare).
 */
import { writeFileSync } from 'fs'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'
import patterns from '../../minimatch/test/patterns.js'
import { minimatch } from '../../minimatch/dist/esm/index.js'

const __dirname = dirname(fileURLToPath(import.meta.url))
const alpha = (a, b) => (a > b ? 1 : a < b ? -1 : 0)
const cases = []

for (const c of patterns) {
  if (typeof c === 'function') {
    c()
    continue
  }
  if (typeof c === 'string') continue
  const pattern = c[0]
  const options = c[2] && typeof c[2] === 'object' ? c[2] : {}
  const f = c[3] || patterns.files
  const actual = minimatch.match([...f], pattern, options || {}).sort(alpha)
  const opts = {}
  for (const [k, v] of Object.entries(options || {})) {
    if (v !== undefined && v !== null) opts[k] = v
  }
  cases.push({ pattern, expect: actual, options: opts, files: [...f] })
}

const out = join(__dirname, 'patterns_node.json')
writeFileSync(out, JSON.stringify(cases))
console.log('wrote', cases.length, 'cases to', out)
