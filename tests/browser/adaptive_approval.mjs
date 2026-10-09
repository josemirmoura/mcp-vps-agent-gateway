import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import fs from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import readline from 'node:readline';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';

const require = createRequire(import.meta.url);
const moduleName = process.env.PORTICO_PLAYWRIGHT_MODULE ||
  (process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES ? path.join(process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES, 'playwright') : 'playwright');
const { chromium, devices } = require(moduleName);
const root = fileURLToPath(new URL('../..', import.meta.url));
const artifacts = process.env.PORTICO_BROWSER_ARTIFACT_DIR || '/tmp/portico-browser-artifacts';
let browser, fixture, metadata;

before(async () => {
  fixture = spawn(process.env.PORTICO_BROWSER_PYTHON || 'python3', ['tests/browser/approval_fixture.py'], {
    cwd: root, stdio: ['ignore', 'pipe', 'inherit'],
  });
  const lines = readline.createInterface({ input: fixture.stdout });
  metadata = await Promise.race([
    once(lines, 'line').then(([line]) => JSON.parse(line)),
    once(fixture, 'exit').then(([code]) => { throw Error('HTTPS fixture exited before startup: ' + code); }),
  ]);
  lines.close();
  browser = await chromium.launch({ headless: true });
  fs.mkdirSync(artifacts, { recursive: true });
}, { timeout: 30_000 });

after(async () => {
  await browser?.close();
  if (fixture && fixture.exitCode === null) {
    const exited = once(fixture, 'exit');
    fixture.kill('SIGTERM');
    await exited;
  }
});

const profiles = [
  { name: 'desktop-chromium', context: { viewport: { width: 1280, height: 900 } } },
  { name: 'mobile-chromium', context: devices['Pixel 7'] },
];

async function run(profile, name, callback) {
  const context = await browser.newContext({ ...profile.context, ignoreHTTPSErrors: true });
  const page = await context.newPage();
  await context.request.get(metadata.host + '/__fixture/reset');
  const uiErrors = [];
  page.on('pageerror', error => uiErrors.push(error.message));
  page.on('dialog', dialog => dialog.accept());
  try {
    await callback({ page, context });
    assert.deepEqual(uiErrors, [], 'UI must not throw uncaught browser exceptions');
  } finally {
    await page.screenshot({ path: path.join(artifacts, profile.name + '-' + name + '.png'), fullPage: true });
    await context.close();
  }
}

async function state(context) {
  return (await context.request.get(metadata.host + '/__fixture/state')).json();
}

async function login(scope) {
  await scope.locator('#login-panel').waitFor({ state: 'visible' });
  await scope.locator('#password').fill(metadata.password);
  await scope.locator('#login-form button').click();
  await scope.locator('#details').waitFor({ state: 'visible' });
  assert.equal(await scope.locator('#password').inputValue(), '', 'Password must clear after submit');
  assert.equal(await scope.locator('#operator').textContent(), 'browser-fixture-owner');
  assert.equal(await scope.locator('#subject').textContent(), 'browser-fixture-agent');
}

async function pending(scope) {
  await scope.locator('#approve:not([disabled])').waitFor();
  await scope.locator('#deny:not([disabled])').waitFor();
}

async function final(scope, decision) {
  await scope.locator('#status').filter({ hasText: decision === 'approve' ? 'Autorização aprovada com sucesso' : 'Solicitação negada com sucesso' }).waitFor();
  assert.equal(await scope.locator('#approve').isDisabled(), true);
  assert.equal(await scope.locator('#deny').isDisabled(), true);
  assert.equal(await scope.locator('#login-panel').isVisible(), false);
}

async function noHorizontalOverflow(scope) {
  const sizes = await scope.locator('body').evaluate(() => ({ viewport: document.documentElement.clientWidth, width: document.documentElement.scrollWidth }));
  assert.ok(sizes.width <= sizes.viewport + 1, JSON.stringify(sizes));
}

for (const profile of profiles) {
  test(profile.name + ': explicit read decisions reuse an authenticated session', { timeout: 30_000 }, () => run(profile, 'portal-session', async ({ page, context }) => {
    await page.goto(metadata.operator + '/operator?request=' + metadata.read);
    await login(page); await pending(page); await noHorizontalOverflow(page);
    assert.equal(await page.locator('#step-up-panel').isVisible(), false);
    assert.equal((await state(context)).decisions.length, 0);
    await page.locator('#approve').click(); await final(page, 'approve');
    await page.reload();
    await page.locator('#status').filter({ hasText: 'Autorização confirmada pelo Broker' }).waitFor();
    assert.equal(await page.locator('#approve').isDisabled(), true);
    await page.goto(metadata.operator + '/operator?request=' + metadata.other);
    await pending(page);
    assert.equal(await page.locator('#login-panel').isVisible(), false);
    assert.equal((await state(context)).decisions.length, 1, 'Session reuse cannot implicitly authorize another request');
    await page.locator('#deny').click(); await final(page, 'deny');
    const observed = await state(context);
    assert.deepEqual(observed.decisions.map(call => call.action), ['approve', 'deny']);
    assert.equal(observed.http.filter(call => call.path.endsWith('/login')).length, 1);
    const cookie = (await context.cookies()).find(value => value.name === 'portico_operator');
    assert.ok(cookie?.secure && cookie?.httpOnly);
    assert.equal(cookie.sameSite, 'Strict');
    assert.equal(cookie.path, '/operator');
    await page.locator('#logout').click();
    await page.locator('#login-panel').waitFor({ state: 'visible' });
    await page.reload();
    await page.locator('#login-panel').waitFor({ state: 'visible' });
  }));

  test(profile.name + ': deny remains final and never grants a pending request', { timeout: 30_000 }, () => run(profile, 'portal-deny', async ({ page, context }) => {
    await page.goto(metadata.operator + '/operator?request=' + metadata.read);
    await login(page); await pending(page);
    await page.locator('#deny').click(); await final(page, 'deny');
    await page.locator('#refresh').click();
    await page.locator('#status').filter({ hasText: 'Solicitação negada. Nenhuma permissão concedida' }).waitFor();
    const observed = await state(context);
    assert.equal(observed.statuses[metadata.read], 'denied');
    assert.deepEqual(observed.decisions.map(call => call.action), ['deny']);
  }));

  test(profile.name + ': critical approval requires fresh per-request password verification', { timeout: 30_000 }, () => run(profile, 'critical-step-up', async ({ page, context }) => {
    await page.goto(metadata.operator + '/operator?request=' + metadata.critical);
    await login(page); await pending(page);
    await page.locator('#step-up-panel').waitFor({ state: 'visible' });
    await page.locator('#step-up-password').fill('wrong-synthetic-passphrase');
    await page.locator('#approve').click();
    await page.locator('#status').filter({ hasText: 'Operador sem autorização ou verificação recusada' }).waitFor();
    assert.equal(await page.locator('#step-up-password').inputValue(), '');
    assert.equal((await state(context)).decisions.length, 0);
    assert.equal(await page.locator('#approve').isDisabled(), true);
    await page.locator('#refresh').click(); await pending(page);
    await page.locator('#step-up-password').fill(metadata.password);
    await page.locator('#approve').click(); await final(page, 'approve');
    const observed = await state(context);
    assert.equal(observed.decisions.length, 1);
    assert.equal(observed.decisions[0].step_up, true);
    assert.deepEqual(observed.http.filter(call => call.path.endsWith('/step-up')).map(call => call.status), [403, 200]);
    assert.equal(await page.locator('#step-up-password').inputValue(), '');
  }));

  test(profile.name + ': critical denial needs no password repetition', { timeout: 30_000 }, () => run(profile, 'critical-deny', async ({ page, context }) => {
    await page.goto(metadata.operator + '/operator?request=' + metadata.sensitive);
    await login(page); await pending(page);
    await page.locator('#step-up-panel').waitFor({ state: 'visible' });
    await page.locator('#deny').click(); await final(page, 'deny');
    const observed = await state(context);
    assert.equal(observed.decisions[0].step_up, false);
    assert.equal(observed.http.filter(call => call.path.endsWith('/step-up')).length, 0);
  }));

  test(profile.name + ': embedded origin isolation, explicit owner decision and Broker-confirmed final UI', { timeout: 30_000 }, () => run(profile, 'apps-embed', async ({ page, context }) => {
    await page.goto(metadata.host + '/host?request=' + metadata.read);
    const appFrame = page.frameLocator('#app');
    await appFrame.locator('#frame').waitFor({ state: 'visible' });
    const portalFrame = appFrame.frameLocator('#frame');
    await login(portalFrame); await pending(portalFrame); await noHorizontalOverflow(portalFrame);
    // The outer AI host and resource cannot read the operator DOM/session.
    const isolation = await page.evaluate(() => {
      try { return document.getElementById('app').contentWindow.document.body.textContent; }
      catch (error) { return error.name; }
    });
    assert.equal(isolation, 'SecurityError');
    const operatorCookie = (await context.cookies()).find(value => value.name === '__Secure-portico_operator_embed');
    assert.ok(operatorCookie?.httpOnly && operatorCookie?.secure);
    assert.equal(operatorCookie.sameSite, 'None');
    assert.equal(operatorCookie.path, '/operator/embed');
    const before = await state(context);
    assert.equal(before.decisions.length, 0);
    await page.evaluate(requestID => window.sendHostMessage({ type: 'portico-operator', request_id: requestID, status: 'approved', password: 'model-supplied' }), metadata.read);
    await page.evaluate(requestID => window.sendHostMessage({ jsonrpc: '2.0', method: 'tools/call', params: { name: 'admin.approval.approve', arguments: { request_id: requestID } } }), metadata.read);
    assert.equal((await state(context)).decisions.length, 0);
    await portalFrame.locator('#approve').click(); await final(portalFrame, 'approve');
    await appFrame.locator('#status').filter({ hasText: 'Broker confirmou a autorização' }).waitFor();
    const messages = await page.evaluate(() => window.bridgeMessages);
    assert.ok(messages.every(message => !JSON.stringify(message).includes(metadata.password)), 'Credentials must never reach host bridge');
    const calls = messages.filter(message => message.method === 'tools/call');
    assert.ok(calls.length > 0);
    assert.ok(calls.every(message => message.params.name === 'permissions.approval_status'));
    // A second request in the same verified host partition reuses the session,
    // while still requiring its own explicit decision.
    await page.goto(metadata.host + '/host?request=' + metadata.other);
    await pending(portalFrame);
    assert.equal(await portalFrame.locator('#login-panel').isVisible(), false);
    assert.equal((await state(context)).decisions.length, 1);
    await portalFrame.locator('#deny').click(); await final(portalFrame, 'deny');
    await appFrame.locator('#status').filter({ hasText: 'Broker confirmou a negação' }).waitFor();
    assert.equal((await state(context)).http.filter(call => call.path.endsWith('/login')).length, 1);
    // An embedded cookie cannot authenticate the independently protected top-level portal.
    await page.goto(metadata.operator + '/operator?request=' + metadata.other);
    await page.locator('#login-panel').waitFor({ state: 'visible' });
  }));

  test(profile.name + ': unverified framing capability retains usable external portal', { timeout: 30_000 }, () => run(profile, 'apps-fallback', async ({ page, context }) => {
    await page.goto(metadata.host + '/host?request=' + metadata.read + '&frame=blocked');
    const appFrame = page.frameLocator('#app');
    await appFrame.locator('#portal').waitFor({ state: 'visible' });
    assert.equal(await appFrame.locator('#frame').isVisible(), false);
    assert.match(await appFrame.locator('#status').textContent(), /não confirmou suporte/);
    const href = await appFrame.locator('#portal').getAttribute('href');
    assert.equal(href, metadata.operator + '/operator?request=' + metadata.read);
    assert.equal((await state(context)).decisions.length, 0);
    await page.goto(href); await login(page); await pending(page);
    await page.locator('#deny').click(); await final(page, 'deny');
  }));
}
