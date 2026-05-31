import { readFileSync } from 'node:fs';
import { parse } from 'yaml';

const workflowFiles = [
  '.github/workflows/ci.yml',
  '.github/workflows/release.yml'
];

const requiredTargets = new Set([
  'windows/amd64',
  'windows/arm64',
  'linux/amd64',
  'linux/arm64',
  'darwin/amd64',
  'darwin/arm64'
]);

for (const file of workflowFiles) {
  const workflow = parse(readFileSync(file, 'utf8'));
  assert(workflow?.env?.GO_VERSION === '1.26.0', `${file}: GO_VERSION must be 1.26.0`);
  assert(workflow?.env?.NODE_VERSION === '24', `${file}: NODE_VERSION must be 24`);
  assertUses(workflow, 'actions/setup-node@v6', file);
  assertUses(workflow, 'actions/setup-go@v6', file);

  const matrixJob = workflow.jobs['build-matrix'] ?? workflow.jobs.build;
  assert(matrixJob, `${file}: missing build matrix job`);
  const targets = new Set(
    matrixJob.strategy.matrix.include.map((entry) => `${entry.goos}/${entry.goarch}`)
  );
  for (const target of requiredTargets) {
    assert(targets.has(target), `${file}: missing ${target}`);
  }
}

console.log('workflow verification ok');

function assertUses(workflow, action, file) {
  for (const job of Object.values(workflow.jobs ?? {})) {
    for (const step of job.steps ?? []) {
      if (step.uses === action) {
        return;
      }
    }
  }
  throw new Error(`${file}: missing ${action}`);
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
