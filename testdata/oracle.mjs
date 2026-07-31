#!/usr/bin/env node
/**
 * Batch oracle for differential fuzzing.
 * stdin:  JSON array of { files: string[], pattern: string, options?: object }
 * stdout: JSON array of string[]  (match results per case)
 *
 * Paths are resolved relative to this file → ../../minimatch
 */
import { createInterface } from 'readline'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'
import { pathToFileURL } from 'url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'minimatch')
const { minimatch } = await import(pathToFileURL(join(root, 'dist/esm/index.js')).href)

const rl = createInterface({ input: process.stdin, crlfDelay: Infinity })
let buf = ''
for await (const line of rl) {
  buf += line
}
const cases = JSON.parse(buf || '[]')
const out = []
for (const c of cases) {
  try {
    const files = Array.isArray(c.files) ? c.files : []
    const pattern = String(c.pattern ?? '')
    const options = c.options && typeof c.options === 'object' ? c.options : {}
    // Guard absurd patterns the same way Go ValidatePattern does roughly
    if (pattern.length > 64 * 1024) {
      out.push({ error: 'pattern is too long' })
      continue
    }
    out.push({ result: minimatch.match(files, pattern, options) })
  } catch (e) {
    out.push({ error: String(e && e.message ? e.message : e) })
  }
}
process.stdout.write(JSON.stringify(out))
