import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve, sep } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const reportPath = resolve(root, 'coverage/coverage-final.json');
const threshold = 95;

if (!existsSync(reportPath)) {
  console.error(`Coverage report not found: ${relative(root, reportPath)}`);
  process.exit(1);
}

function sourceFiles(directory) {
  return readdirSync(directory).flatMap((name) => {
    const path = resolve(directory, name);
    return statSync(path).isDirectory() ? sourceFiles(path) : [path];
  });
}

function sourcePath(path) {
  const marker = `${sep}src${sep}`;
  const index = path.lastIndexOf(marker);
  return index === -1 ? null : path.slice(index + 1).split(sep).join('/');
}

function covered(values) {
  const counts = Object.values(values).flat();
  return { covered: counts.filter((count) => count > 0).length, total: counts.length };
}

const report = JSON.parse(readFileSync(reportPath, 'utf8'));
const entries = new Map(
  Object.values(report)
    .map((entry) => [sourcePath(entry.path), entry])
    .filter(([path]) => path !== null),
);
// Bootstrap, test harness, and a type-only module have no independently testable runtime behavior.
const nonInstrumented = new Set([
  'src/main.ts',
  'src/test-setup.ts',
  'src/app/core/models/auth.model.ts',
]);
const relevant = sourceFiles(resolve(root, 'src'))
  .map((path) => relative(root, path).split(sep).join('/'))
  .filter((path) => path.endsWith('.ts'))
  .filter((path) => !path.endsWith('.spec.ts'))
  .filter((path) => !nonInstrumented.has(path));
const missing = relevant.filter((path) => !entries.has(path));

const totals = {
  statements: { covered: 0, total: 0 },
  branches: { covered: 0, total: 0 },
  functions: { covered: 0, total: 0 },
  lines: { covered: 0, total: 0 },
};

for (const path of relevant) {
  const entry = entries.get(path);
  if (!entry) continue;
  const metrics = {
    statements: covered(entry.s),
    branches: covered(entry.b),
    functions: covered(entry.f),
    lines: covered(entry.s),
  };
  for (const [name, value] of Object.entries(metrics)) {
    totals[name].covered += value.covered;
    totals[name].total += value.total;
  }
}

let failed = missing.length > 0;
console.log(`source files: ${relevant.length - missing.length}/${relevant.length}`);
if (missing.length > 0) {
  console.error(`Missing relevant source files (${missing.length}):`);
  for (const path of missing) console.error(`  ${path}`);
}

for (const [name, { covered: count, total }] of Object.entries(totals)) {
  const percentage = total === 0 ? 100 : (count / total) * 100;
  console.log(`${name}: ${percentage.toFixed(2)}% (${count}/${total})`);
  failed ||= percentage < threshold;
}

if (failed) {
  console.error(`Coverage gate failed: every metric must be >= ${threshold}% and every relevant source file must appear.`);
  process.exit(1);
}
