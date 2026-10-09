import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

// This is a protocol harness, not evidence that a vendor renders MCP Apps.
const source = fs.readFileSync(new URL('../../internal/gateway/approval_app.html', import.meta.url), 'utf8');
const script = source.match(/<script>([\s\S]*?)<\/script>/)[1];
const requestId = 'apr_abcdefgh1234';
const operatorOrigin = 'https://operator.test';
const hostOrigin = 'https://host.test';

function harness({ embedded = true } = {}) {
  const sent = [], listeners = {}, timers = new Map(), elements = new Map();
  let timerID = 0;
  for (const id of ['status', 'frame', 'portal', 'portal-url', 'refresh', 'cli']) {
    elements.set(id, {
      hidden: ['frame', 'portal'].includes(id), disabled: id === 'refresh',
      textContent: '', src: '', href: '', contentWindow: {}, handlers: {},
      addEventListener(type, handler) { this.handlers[type] = handler; },
    });
  }
  const parent = { postMessage(value, target) { sent.push({ value: JSON.parse(JSON.stringify(value)), target }); } };
  const window = { addEventListener(type, handler) { listeners[type] = handler; } };
  const context = vm.createContext({
    window, parent, URL, Map, Promise, Error,
    document: { getElementById(id) { return elements.get(id); } },
    setTimeout(fn, ms) { const id = ++timerID; timers.set(id, { fn, ms }); return id; },
    clearTimeout(id) { timers.delete(id); },
  });
  vm.runInContext(script.replace('__PORTICO_CONFIG__', JSON.stringify({ operatorOrigin, embedded })), context);
  function message(value, { origin = hostOrigin, source = parent } = {}) {
    listeners.message({ data: value, origin, source });
  }
  function initialize(caps = { sandbox: { csp: { frameDomains: [operatorOrigin] } } }, version = '2026-01-26') {
    const id = sent.find(x => x.value.method === 'ui/initialize').value.id;
    message({ jsonrpc: '2.0', id, result: { protocolVersion: version, hostCapabilities: caps } });
  }
  function toolResult(overrides = {}) {
    message({ jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: { structuredContent: {
      request_id: requestId, operator_approval_url: operatorOrigin + '/operator?request=' + requestId, ...overrides,
    } } });
  }
  function timeout(ms) {
    for (const [id, timer] of [...timers]) if (timer.ms === ms) { timers.delete(id); timer.fn(); }
  }
  return { sent, elements, parent, window, message, initialize, toolResult, timeout };
}

test('Apps handshake uses a credential-free protocol and explicit framing capability', () => {
  const h = harness();
  assert.equal(h.sent[0].value.method, 'ui/initialize');
  assert.equal(h.sent[0].target, '*');
  h.initialize(); h.toolResult();
  assert.equal(h.sent[1].value.method, 'ui/notifications/initialized');
  assert.equal(h.sent[1].target, hostOrigin);
  assert.equal(h.elements.get('frame').src, operatorOrigin + '/operator/embed?request=' + requestId);
  assert.equal(h.elements.get('frame').hidden, false);
  assert.equal(h.elements.get('portal').hidden, false);
  assert.equal(h.elements.get('portal').href, operatorOrigin + '/operator?request=' + requestId);
});

test('missing, unverified or wrong framing capability always leaves portal fallback', () => {
  for (const caps of [{}, { sandbox: { csp: { frameDomains: ['https://elsewhere.test'] } } }, { sandbox: { csp: { frameDomains: operatorOrigin } } }]) {
    const h = harness(); h.initialize(caps); h.toolResult();
    assert.equal(h.elements.get('frame').hidden, true);
    assert.equal(h.elements.get('portal').hidden, false);
    assert.match(h.elements.get('status').textContent, /não confirmou suporte/);
  }
  const disabled = harness({ embedded: false }); disabled.initialize(); disabled.toolResult();
  assert.equal(disabled.elements.get('frame').hidden, true);
  const wrongVersion = harness(); wrongVersion.initialize({}, 'unverified'); wrongVersion.toolResult();
  assert.equal(wrongVersion.elements.get('frame').hidden, true);
  assert.equal(wrongVersion.elements.get('portal').hidden, false);
});

test('unresponsive bridge and blocked iframe explain the fallback', () => {
  const bridge = harness(); bridge.timeout(8000);
  assert.match(bridge.elements.get('status').textContent, /Bridge MCP Apps indisponível/);
  const frame = harness(); frame.initialize(); frame.toolResult(); frame.timeout(6000);
  assert.match(frame.elements.get('status').textContent, /bloqueou a Central/);
  assert.equal(frame.elements.get('portal').hidden, false);
  assert.match(frame.elements.get('cli').textContent, /scripts\/operator-approvals.py --request apr_/);
});

test('host response is accepted only from the parent and pins subsequent origin', () => {
  const h = harness();
  const id = h.sent[0].value.id;
  h.message({ jsonrpc: '2.0', id, result: { protocolVersion: '2026-01-26', hostCapabilities: {} } }, { source: {} });
  assert.equal(h.sent.length, 1);
  h.initialize();
  h.message({ jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: { structuredContent: {
    request_id: requestId, operator_approval_url: operatorOrigin + '/operator?request=' + requestId,
  } } }, { origin: 'https://spoof.test' });
  assert.equal(h.elements.get('portal').hidden, true);
  h.toolResult();
  assert.equal(h.elements.get('portal').hidden, false);
});

test('tool output cannot redirect owner credentials to an arbitrary URL', () => {
  const invalid = [
    'https://attacker.test/operator?request=' + requestId,
    operatorOrigin + '/operator/embed?request=' + requestId,
    operatorOrigin + '/operator?request=apr_different1234',
    operatorOrigin + '/operator?request=' + requestId + '#token',
    'https://owner:password@operator.test/operator?request=' + requestId,
    'javascript:alert(1)',
  ];
  for (const link of invalid) {
    const h = harness(); h.initialize(); h.toolResult({ operator_approval_url: link });
    assert.equal(h.elements.get('portal').hidden, true, link);
    assert.equal(h.elements.get('frame').hidden, true, link);
  }
  const h = harness(); h.initialize(); h.toolResult({ request_id: 'not-a-request' });
  assert.equal(h.elements.get('portal').hidden, true);
});

test('operator status messages require exact source, origin and request', () => {
  const h = harness(); h.initialize(); h.toolResult();
  const initial = h.elements.get('status').textContent;
  const ready = { type: 'portico-operator', request_id: requestId, status: 'ready' };
  h.message(ready, { origin: operatorOrigin, source: {} });
  h.message(ready, { origin: 'https://spoof.test', source: h.elements.get('frame').contentWindow });
  h.message({ ...ready, request_id: 'apr_different1234' }, { origin: operatorOrigin, source: h.elements.get('frame').contentWindow });
  assert.equal(h.elements.get('status').textContent, initial);
  h.message(ready, { origin: operatorOrigin, source: h.elements.get('frame').contentWindow });
  assert.match(h.elements.get('status').textContent, /Central conectada/);
});

test('frame approval announcement only queries Broker state; it never grants permission', async () => {
  const h = harness(); h.initialize(); h.toolResult();
  h.message({ type: 'portico-operator', request_id: requestId, status: 'approved' }, {
    origin: operatorOrigin, source: h.elements.get('frame').contentWindow,
  });
  const call = h.sent.at(-1).value;
  assert.equal(call.method, 'tools/call');
  assert.equal(call.params.name, 'permissions.approval_status');
  assert.deepEqual(call.params.arguments, { request_id: requestId });
  h.message({ jsonrpc: '2.0', id: call.id, result: { structuredContent: { request_id: requestId, status: 'pending' } } });
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.elements.get('status').textContent, 'Aguardando decisão do operador.');
});

test('no parent message can invoke approval or inject credentials', () => {
  const h = harness(); h.initialize(); h.toolResult();
  const before = h.sent.length;
  for (const value of [
    { type: 'portico-operator', request_id: requestId, status: 'approved', password: 'secret' },
    { jsonrpc: '2.0', method: 'tools/call', params: { name: 'admin.approval.approve', arguments: { request_id: requestId } } },
    { jsonrpc: '2.0', method: 'ui/notifications/approval', params: { decision: 'approve', password: 'secret' } },
  ]) h.message(value);
  assert.equal(h.sent.length, before);
  assert.ok(h.sent.every(x => !JSON.stringify(x).includes('secret')));
});

test('final UI shows only Broker-confirmed decision and refuses mismatched results', async () => {
  for (const [state, text] of [['approved', /Broker confirmou a autorização/], ['denied', /Broker confirmou a negação/], ['expired', /expirou/], ['revoked', /revogada/]]) {
    const h = harness(); h.initialize(); h.toolResult();
    const promise = h.elements.get('refresh').handlers.click();
    const call = h.sent.at(-1).value;
    h.message({ jsonrpc: '2.0', id: call.id, result: { structuredContent: { request_id: requestId, status: state } } });
    await promise;
    assert.match(h.elements.get('status').textContent, text);
  }
  const h = harness(); h.initialize(); h.toolResult();
  const promise = h.elements.get('refresh').handlers.click();
  const call = h.sent.at(-1).value;
  h.message({ jsonrpc: '2.0', id: call.id, result: { structuredContent: { request_id: 'apr_different1234', status: 'approved' } } });
  await promise;
  assert.match(h.elements.get('status').textContent, /Broker não confirmou/);
});

test('cancellation cannot produce an approval and reports no automatic decision', () => {
  const h = harness(); h.initialize(); h.toolResult();
  h.message({ jsonrpc: '2.0', method: 'ui/notifications/tool-cancelled' });
  assert.match(h.elements.get('status').textContent, /Nenhuma decisão automática/);
  assert.ok(h.sent.every(x => !x.value.params?.name?.includes('approve')));
});

test('an explicit owner link click uses verified host openLinks capability only', async () => {
  const h = harness(); h.initialize({ openLinks: {} }); h.toolResult();
  assert.equal(h.sent.some(x => x.value.method === 'ui/open-link'), false);
  let prevented = false;
  const promise = h.elements.get('portal').handlers.click({ preventDefault() { prevented = true; } });
  const call = h.sent.at(-1).value;
  assert.equal(prevented, true);
  assert.equal(call.method, 'ui/open-link');
  assert.deepEqual(call.params, { url: operatorOrigin + '/operator?request=' + requestId });
  h.message({ jsonrpc: '2.0', id: call.id, result: {} });
  await promise;
  const unsupported = harness(); unsupported.initialize({}); unsupported.toolResult();
  const count = unsupported.sent.length;
  await unsupported.elements.get('portal').handlers.click({ preventDefault() { throw Error('Do not suppress the raw fallback'); } });
  assert.equal(unsupported.sent.length, count);
  assert.equal(unsupported.elements.get('portal-url').textContent, operatorOrigin + '/operator?request=' + requestId);
});

test('a late status response cannot overwrite a different request in a reused app', async () => {
  const h = harness(); h.initialize(); h.toolResult();
  const promise = h.elements.get('refresh').handlers.click();
  const call = h.sent.at(-1).value;
  const second = 'apr_different1234';
  h.toolResult({ request_id: second, operator_approval_url: operatorOrigin + '/operator?request=' + second });
  const before = h.elements.get('status').textContent;
  h.message({ jsonrpc: '2.0', id: call.id, result: { structuredContent: { request_id: requestId, status: 'approved' } } });
  await promise;
  assert.equal(h.elements.get('status').textContent, before);
  assert.equal(h.elements.get('portal').href, operatorOrigin + '/operator?request=' + second);
});
