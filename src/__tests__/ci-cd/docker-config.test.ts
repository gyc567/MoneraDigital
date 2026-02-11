/**
 * CI/CD Configuration Tests
 *
 * Tests for Docker, GitHub Actions, and deployment configurations.
 * Following TDD: Tests written first, implementation follows.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import fs from 'fs';
import path from 'path';
import yaml from 'js-yaml';

const CI_CD_ROOT = path.resolve(__dirname, '../../..');

// ============================================
// Dockerfile Tests
// ============================================

describe('Dockerfile Configuration', () => {
  let dockerfileContent: string;

  beforeAll(() => {
    const dockerfilePath = path.join(CI_CD_ROOT, 'Dockerfile');
    if (fs.existsSync(dockerfilePath)) {
      dockerfileContent = fs.readFileSync(dockerfilePath, 'utf-8');
    }
  });

  it('Dockerfile should exist', () => {
    const dockerfilePath = path.join(CI_CD_ROOT, 'Dockerfile');
    expect(fs.existsSync(dockerfilePath), 'Dockerfile should exist').toBe(true);
  });

  it('Dockerfile should use multi-stage build', () => {
    expect(dockerfileContent).toContain('FROM');
    expect(dockerfileContent).toContain('AS');
    // Should have at least 2 stages (builder and runtime)
    const stageCount = (dockerfileContent.match(/FROM\s+.*\s+AS\s+/gi) || []).length;
    expect(stageCount, 'Multi-stage build should be used').toBeGreaterThanOrEqual(2);
  });

  it('Dockerfile should use alpine base images', () => {
    expect(dockerfileContent).toMatch(/FROM\s+node:[\d.]+-alpine/i);
    expect(dockerfileContent).toMatch(/FROM\s+golang:[\d.]+-alpine/i);
    expect(dockerfileContent).toMatch(/FROM\s+alpine:[\d.]+/i);
  });

  it('Dockerfile should set correct working directory', () => {
    expect(dockerfileContent).toContain('WORKDIR /app');
  });

  it('Dockerfile should expose correct port', () => {
    expect(dockerfileContent).toContain('EXPOSE 5000');
  });

  it('Dockerfile should use non-root user', () => {
    expect(dockerfileContent).toContain('adduser');
    expect(dockerfileContent).toContain('USER app');
  });

  it('Dockerfile should include health check', () => {
    expect(dockerfileContent.toLowerCase()).toContain('healthcheck');
  });

  it('Dockerfile should set GIN_MODE to release', () => {
    expect(dockerfileContent).toContain('GIN_MODE=release');
  });

  it('Dockerfile should use static Go binary (CGO_ENABLED=0)', () => {
    expect(dockerfileContent).toContain('CGO_ENABLED=0');
  });

  it('Dockerfile should copy frontend build artifacts', () => {
    expect(dockerfileContent).toContain('COPY --from=');
    expect(dockerfileContent).toContain('dist');
  });

  it('Dockerfile should not contain development dependencies', () => {
    expect(dockerfileContent).not.toContain('npm run dev');
    expect(dockerfileContent).not.toContain('vite');
  });
});

// ============================================
// .dockerignore Tests
// ============================================

describe('.dockerignore Configuration', () => {
  let dockerignoreContent: string;

  beforeAll(() => {
    const dockerignorePath = path.join(CI_CD_ROOT, '.dockerignore');
    if (fs.existsSync(dockerignorePath)) {
      dockerignoreContent = fs.readFileSync(dockerignorePath, 'utf-8');
    }
  });

  it('.dockerignore should exist', () => {
    const dockerignorePath = path.join(CI_CD_ROOT, '.dockerignore');
    expect(fs.existsSync(dockerignorePath), '.dockerignore should exist').toBe(true);
  });

  it('.dockerignore should exclude Git files', () => {
    expect(dockerignoreContent).toContain('.git');
    expect(dockerignoreContent).toContain('.gitignore');
  });

  it('.dockerignore should exclude IDE files', () => {
    expect(dockerignoreContent).toContain('.vscode');
    expect(dockerignoreContent).toContain('.idea');
  });

  it('.dockerignore should exclude node_modules', () => {
    expect(dockerignoreContent).toContain('node_modules');
  });

  it('.dockerignore should exclude test files', () => {
    expect(dockerignoreContent).toContain('*.test.ts');
    expect(dockerignoreContent).toContain('*_test.go');
  });

  it('.dockerignore should exclude documentation', () => {
    expect(dockerignoreContent).toContain('README.md');
  });

  it('.dockerignore should include dist folder', () => {
    expect(dockerignoreContent).toContain('dist/');
  });

  it('.dockerignore should exclude OS files', () => {
    expect(dockerignoreContent).toContain('.DS_Store');
    expect(dockerignoreContent).toContain('Thumbs.db');
  });
});

// ============================================
// docker-compose.yml Tests
// ============================================

describe('docker-compose.yml Configuration', () => {
  let composeConfig: Record<string, unknown>;
  let composeContent: string;

  beforeAll(() => {
    const composePath = path.join(CI_CD_ROOT, 'docker-compose.yml');
    if (fs.existsSync(composePath)) {
      composeContent = fs.readFileSync(composePath, 'utf-8');
      composeConfig = yaml.load(composeContent) as Record<string, unknown>;
    }
  });

  it('docker-compose.yml should exist', () => {
    const composePath = path.join(CI_CD_ROOT, 'docker-compose.yml');
    expect(fs.existsSync(composePath), 'docker-compose.yml should exist').toBe(true);
  });

  it('docker-compose.yml should be valid YAML', () => {
    expect(composeConfig).toBeDefined();
    expect(composeConfig).not.toBeNull();
  });

  it('docker-compose.yml should define app service', () => {
    expect(composeConfig).toHaveProperty('services');
    expect((composeConfig.services as Record<string, unknown>)?.app).toBeDefined();
  });

  it('app service should have image configured', () => {
    const services = composeConfig.services as Record<string, Record<string, unknown>>;
    const appService = services?.app;
    expect(appService).toHaveProperty('image');
  });

  it('app service should expose port 5000', () => {
    const services = composeConfig.services as Record<string, Record<string, unknown>>;
    const appService = services?.app;
    expect(appService).toHaveProperty('ports');
    const ports = appService.ports as string[];
    expect(ports?.join(' ')).toContain('5000');
  });

  it('app service should have environment variables', () => {
    const services = composeConfig.services as Record<string, Record<string, unknown>>;
    const appService = services?.app;
    expect(appService).toHaveProperty('environment');
  });

  it('app service should have restart policy', () => {
    const services = composeConfig.services as Record<string, Record<string, unknown>>;
    const appService = services?.app;
    expect(appService).toHaveProperty('restart');
    expect(appService.restart).toBe('unless-stopped');
  });

  it('app service should have healthcheck', () => {
    const services = composeConfig.services as Record<string, Record<string, unknown>>;
    const appService = services?.app;
    expect(appService).toHaveProperty('healthcheck');
  });
});

// ============================================
// GitHub Actions Workflow Tests
// ============================================

describe('GitHub Actions Workflow Configuration', () => {
  let workflowContent: string;
  let workflowConfig: Record<string, unknown>;
  let workflowPath: string;

  beforeAll(() => {
    workflowPath = path.join(CI_CD_ROOT, '.github/workflows/deploy.yml');
    if (fs.existsSync(workflowPath)) {
      workflowContent = fs.readFileSync(workflowPath, 'utf-8');
      workflowConfig = yaml.load(workflowContent) as Record<string, unknown>;
    }
  });

  it('deploy.yml workflow should exist', () => {
    const workflowPath = path.join(CI_CD_ROOT, '.github/workflows/deploy.yml');
    expect(fs.existsSync(workflowPath), 'deploy.yml should exist').toBe(true);
  });

  it('workflow should have name', () => {
    expect(workflowConfig).toHaveProperty('name');
  });

  it('workflow should trigger on push to main', () => {
    expect(workflowConfig).toHaveProperty('on');
    const onConfig = workflowConfig.on as Record<string, unknown>;
    expect(onConfig).toHaveProperty('push');
    const pushConfig = onConfig.push as Record<string, unknown>;
    expect((pushConfig.branches as string[]).includes('main')).toBe(true);
  });

  it('workflow should have build job', () => {
    expect(workflowConfig).toHaveProperty('jobs');
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    expect(jobs).toHaveProperty('build');
  });

  it('build job should run on ubuntu-latest', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const buildJob = jobs.build as Record<string, unknown>;
    expect(buildJob).toHaveProperty('runs-on');
    expect(buildJob['runs-on']).toBe('ubuntu-latest');
  });

  it('workflow should use Docker Buildx', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const buildJob = jobs.build as Record<string, unknown>;
    const steps = buildJob.steps as Record<string, unknown>[];
    const hasBuildx = steps.some(step =>
      (step.uses as string)?.includes('docker/setup-buildx-action')
    );
    expect(hasBuildx, 'Should use Docker Buildx').toBe(true);
  });

  it('workflow should login to GHCR', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const buildJob = jobs.build as Record<string, unknown>;
    const steps = buildJob.steps as Record<string, unknown>[];
    const hasLogin = steps.some(step =>
      (step.uses as string)?.includes('docker/login-action')
    );
    expect(hasLogin, 'Should login to container registry').toBe(true);
  });

  it('workflow should build and push Docker image', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const buildJob = jobs.build as Record<string, unknown>;
    const steps = buildJob.steps as Record<string, unknown>[];
    const hasPush = steps.some(step =>
      (step.uses as string)?.includes('docker/build-push-action')
    );
    expect(hasPush, 'Should build and push Docker image').toBe(true);
  });

  it('workflow should have deploy job', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    expect(jobs).toHaveProperty('deploy');
    const deployJob = jobs.deploy as Record<string, unknown>;
    expect(deployJob).toHaveProperty('needs');
    expect((deployJob.needs as string[]).includes('build')).toBe(true);
  });

  it('deploy job should use SSH action', () => {
    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const deployJob = jobs.deploy as Record<string, unknown>;
    const steps = deployJob.steps as Record<string, unknown>[];
    const hasSSH = steps.some(step =>
      (step.uses as string)?.includes('ssh-action')
    );
    expect(hasSSH, 'Should use SSH action for deployment').toBe(true);
  });

  it('workflow should not expose secrets in logs', () => {
    const workflowStr = fs.readFileSync(workflowPath, 'utf-8');
    expect(workflowStr).not.toContain('REPLIT_SSH_KEY=');
    expect(workflowStr).not.toContain('DATABASE_URL=');
    expect(workflowStr).not.toContain('your-ssh-key');
    expect(workflowStr).not.toContain('your-database-url');
  });
});

// ============================================
// Deployment Script Tests
// ============================================

describe('Deployment Script Configuration', () => {
  let deployScriptContent: string;

  beforeAll(() => {
    const scriptPath = path.join(CI_CD_ROOT, 'scripts/deploy.sh');
    if (fs.existsSync(scriptPath)) {
      deployScriptContent = fs.readFileSync(scriptPath, 'utf-8');
    }
  });

  it('deploy.sh should exist', () => {
    const scriptPath = path.join(CI_CD_ROOT, 'scripts/deploy.sh');
    expect(fs.existsSync(scriptPath), 'deploy.sh should exist').toBe(true);
  });

  it('deploy.sh should be executable', () => {
    const scriptPath = path.join(CI_CD_ROOT, 'scripts/deploy.sh');
    const stat = fs.statSync(scriptPath);
    const mode = stat.mode & 0o111;
    expect(mode, 'Script should be executable').not.toBe(0);
  });

  it('deploy.sh should pull latest image', () => {
    expect(deployScriptContent).toContain('docker pull');
  });

  it('deploy.sh should use docker compose', () => {
    expect(deployScriptContent).toContain('docker compose');
  });

  it('deploy.sh should restart containers', () => {
    expect(deployScriptContent).toContain('docker compose up -d');
  });

  it('deploy.sh should verify deployment', () => {
    expect(deployScriptContent).toContain('curl');
    expect(deployScriptContent).toContain('health');
  });

  it('deploy.sh should have error handling', () => {
    expect(deployScriptContent).toContain('set -e');
  });
});

// ============================================
// Integration Tests
// ============================================

describe('CI/CD Integration', () => {
  it('all required files should exist', () => {
    const requiredFiles = [
      'Dockerfile',
      '.dockerignore',
      'docker-compose.yml',
      '.github/workflows/deploy.yml',
      'scripts/deploy.sh',
    ];

    for (const file of requiredFiles) {
      const filePath = path.join(CI_CD_ROOT, file);
      expect(fs.existsSync(filePath), `${file} should exist`).toBe(true);
    }
  });

  it('Dockerfile and docker-compose should use same port', () => {
    const dockerfilePath = path.join(CI_CD_ROOT, 'Dockerfile');
    const composePath = path.join(CI_CD_ROOT, 'docker-compose.yml');

    const dockerfileContent = fs.readFileSync(dockerfilePath, 'utf-8');
    const composeContent = fs.readFileSync(composePath, 'utf-8');

    // Both should reference port 5000
    expect(dockerfileContent).toContain('EXPOSE 5000');
    expect(composeContent).toContain('5000:5000');
  });

  it('GitHub Actions should reference correct image name', () => {
    const workflowPath = path.join(CI_CD_ROOT, '.github/workflows/deploy.yml');
    const workflowContent = fs.readFileSync(workflowPath, 'utf-8');
    const workflowConfig = yaml.load(workflowContent) as Record<string, unknown>;

    const jobs = workflowConfig.jobs as Record<string, unknown>;
    const buildJob = jobs.build as Record<string, unknown>;
    const steps = buildJob.steps as Record<string, unknown>[];

    const metaStep = steps.find(step => step.id === 'meta');
    expect(metaStep, 'Should have metadata step').toBeDefined();
  });
});
