// Run with: node --test scripts/settings-ui.test.cjs
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

class Control {
  constructor() { this.listeners = {}; this.dataset = {}; this.value = ''; this.checked = false; this.disabled = false; this.textContent = ''; this.innerHTML = ''; }
  addEventListener(name, callback) { (this.listeners[name] ||= []).push(callback); }
  emit(name) { for (const callback of this.listeners[name] || []) callback(); }
  setAttribute(name, value) { this[name] = value; }
  removeAttribute(name) { delete this[name]; }
}
function setup(responseFactory, provider = 'tmdb') {
  const publicProvider = provider === 'musicbrainz' || provider === 'openlibrary';
  const button = new Control(), input = new Control(), result = new Control(), remove = new Control();
  button.dataset = { testProvider: provider, testUrl: '/en/settings/providers/test' };
  button.form = publicProvider ? null : { querySelector: () => remove };
  result.dataset = { testing: 'Testing', error: 'Unavailable' };
  const requests = [];
  const document = {
    querySelectorAll: () => [button],
    getElementById: id => id === 'provider-' + provider ? (publicProvider ? null : input) : result,
  };
  const context = {
    document, URLSearchParams, AbortController, window: { addEventListener() {} },
    fetch: async (url, options) => {
      requests.push({ url, options });
      return responseFactory ? responseFactory() : { ok: true, text: async () => '<span>Key accepted</span>' };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../internal/static/public/js/settings.js'), 'utf8'), context);
  return { button, input, result, remove, requests };
}
const flush = () => new Promise(resolve => setImmediate(resolve));

test('Test sends only this provider’s candidate to the test endpoint without submitting Save', async () => {
  const { button, input, result, requests } = setup();
  input.value = 'candidate-secret';
  button.emit('click');
  assert.equal(button.disabled, true);
  assert.equal(result.textContent, 'Testing');
  await flush();
  assert.equal(requests[0].url, '/en/settings/providers/test');
  assert.equal(requests[0].options.method, 'POST');
  assert.equal(requests[0].options.body.get('provider'), 'tmdb');
  assert.equal(requests[0].options.body.get('key'), 'candidate-secret');
  assert.match(result.innerHTML, /Key accepted/);
  assert.equal(button.disabled, false);
  assert.equal(input.value, 'candidate-secret');
});

test('editing a candidate cancels an in-flight check and ignores its stale response', async () => {
  let resolve;
  const { button, input, result, requests } = setup(() => new Promise(r => { resolve = r; }));
  button.emit('click');
  input.value = 'replacement'; input.emit('input');
  assert.equal(requests[0].options.signal.aborted, true);
  assert.equal(result.textContent, '');
  resolve({ ok: true, text: async () => '<span>Stale result</span>' }); await flush();
  assert.equal(result.innerHTML, '');
  assert.equal(button.disabled, false);
});

test('blank candidate and Remove flag let the server test the pending server default', async () => {
  const { button, remove, requests } = setup();
  remove.checked = true;
  button.emit('click'); await flush();
  assert.equal(requests[0].options.body.get('key'), '');
  assert.equal(requests[0].options.body.get('remove'), 'on');
});

test('connection failure displays a localized fallback and allows retry', async () => {
  const { button, result, requests } = setup(() => { throw new Error('private transport detail'); });
  button.emit('click'); await flush();
  assert.equal(result.textContent, 'Unavailable');
  assert.equal(button.disabled, false);
  button.emit('click'); await flush();
  assert.equal(requests.length, 2);
});

test('public provider connection buttons work without a form or key input', async () => {
  for (const provider of ['musicbrainz', 'openlibrary']) {
    const { button, result, requests } = setup(() => ({ ok: true, text: async () => '<span>Connection successful</span>' }), provider);
    button.emit('click');
    assert.equal(button.disabled, true);
    await flush();
    assert.equal(requests[0].options.body.get('provider'), provider);
    assert.equal(requests[0].options.body.has('key'), false);
    assert.equal(requests[0].options.body.has('remove'), false);
    assert.match(result.innerHTML, /Connection successful/);
    assert.equal(button.disabled, false);
  }
});
