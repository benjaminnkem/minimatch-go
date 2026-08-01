#!/usr/bin/env node
/**
 * Comparative bench — Node/isaacs minimatch side.
 * stdout: JSON object with scenarios + meta.
 */
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'
import { pathToFileURL } from 'url'
import { performance } from 'perf_hooks'
import { spawnSync } from 'child_process'

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'minimatch')
const { minimatch, Minimatch, braceExpand } = await import(
  pathToFileURL(join(root, 'dist/esm/index.js')).href
)

const patterns = ['*', '**', 'a/**/b', '*.js', '[a-z]*', '?(a|b)', '!(x)', '{a,b}*']
const files = ['a', 'b', 'abc', 'foo.js', 'a/b/c', 'x']
const opts = { platform: 'linux' }

function quantiles(samples) {
  if (!samples.length) return { p50: 0, p95: 0, p99: 0, mean: 0 }
  const s = [...samples].sort((a, b) => a - b)
  const q = (p) => s[Math.min(s.length - 1, Math.floor(p * (s.length - 1)))]
  const mean = s.reduce((a, b) => a + b, 0) / s.length
  return { p50: q(0.5), p95: q(0.95), p99: q(0.99), mean }
}

function timeLoop(fn, iters, sampleEvery = 1) {
  // warmup
  for (let i = 0; i < Math.min(200, iters); i++) fn()
  const samples = []
  const t0 = performance.now()
  for (let i = 0; i < iters; i++) {
    const a = performance.now()
    fn()
    if (i % sampleEvery === 0) samples.push((performance.now() - a) * 1e6) // ns
  }
  const totalMs = performance.now() - t0
  const q = quantiles(samples)
  return {
    ops: iters,
    ns_per_op_mean: (totalMs * 1e6) / iters,
    ns_per_op_p50: q.p50,
    ns_per_op_p95: q.p95,
    ns_per_op_p99: q.p99,
    samples: samples.length,
  }
}

// startup measured by parent via separate spawn; stub here
const scenarios = {}

scenarios.match_oneshot = timeLoop(() => {
  minimatch('foo/bar.js', '**/*.js', opts)
}, 3000)

const mm = new Minimatch('**/*.js', opts)
scenarios.match_compiled = timeLoop(() => {
  mm.match('foo/bar/baz.js')
}, 20000)

const mmExt = new Minimatch('*(a|b|c)', opts)
scenarios.match_extglob = timeLoop(() => {
  mmExt.match('abc')
}, 20000)

scenarios.match_complex_compile = timeLoop(() => {
  new Minimatch('a/{b,c}/**/!(tmp)/*.{js,ts}', { ...opts, nonegate: true })
}, 500)

scenarios.brace_expand = timeLoop(() => {
  braceExpand('a{1..5}{b,c,d}x')
}, 5000)

const compiled = patterns.map((p) => new Minimatch(p, { ...opts, nonegate: true }))
scenarios.corpus = timeLoop(() => {
  for (const m of compiled) {
    for (const f of files) m.match(f)
  }
}, 2000)

const mem = process.memoryUsage()
const out = {
  impl: 'node-isaacs-minimatch',
  version: '10.2.6',
  meta: {
    node: process.version,
    platform: process.platform,
    arch: process.arch,
  },
  memory: {
    rss_bytes: mem.rss,
    heap_bytes: mem.heapUsed,
  },
  scenarios,
}

process.stdout.write(JSON.stringify(out, null, 2))
