// Contract tests for contracts/openapi/portal-api-v1.yaml (Phase 4 Group 3; owner decisions G3-1 to
// G3-6). Redocly checks validity and style; these tests check the security rules of the portal API
// and that it stays consistent with the public API. They read the bundled specifications, so run
// `pnpm run bundle` first (`pnpm run check` does both). Standard library only (node:test).

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const root = new URL('..', import.meta.url);
function load(name) {
  const path = fileURLToPath(new URL(`build/${name}.json`, root));
  if (!existsSync(path)) {
    throw new Error(`${path} not found: run "pnpm run bundle" first (or "pnpm run check")`);
  }
  return JSON.parse(readFileSync(path, 'utf8'));
}
const spec = load('portal-api-v1');
const publicSpec = load('public-api-v1');

const METHODS = ['get', 'put', 'post', 'delete', 'patch'];
function operations() {
  const ops = [];
  for (const [path, item] of Object.entries(spec.paths)) {
    for (const method of METHODS) {
      if (item[method]) ops.push({ path, method, op: item[method], id: `${method.toUpperCase()} ${path}` });
    }
  }
  return ops;
}
function resolve(node) {
  let n = node;
  while (n && n.$ref) {
    n = n.$ref.replace(/^#\//, '').split('/').reduce((acc, key) => acc[key], spec);
  }
  return n;
}
function headerParams(op) {
  return (op.parameters ?? []).map(resolve).filter((p) => p.in === 'header');
}
const isSignIn = ({ path, method }) => path === '/portal/v1/session' && method === 'post';

test('every path is under /portal/v1 and none is a public API path', () => {
  for (const path of Object.keys(spec.paths)) {
    assert.match(path, /^\/portal\/v1\//, path);
  }
});

test('the session cookie authenticates every operation except sign-in', () => {
  assert.deepEqual(spec.security, [{ portal_session: [] }]);
  const scheme = spec.components.securitySchemes.portal_session;
  assert.equal(scheme.type, 'apiKey');
  assert.equal(scheme.in, 'cookie');
  assert.equal(scheme.name, '__Host-sms_session');
  for (const o of operations()) {
    if (isSignIn(o)) {
      assert.deepEqual(o.op.security, [], 'sign-in must not require a session');
    } else {
      assert.equal(o.op.security, undefined, `${o.id} must use the global session security`);
      assert.ok(o.op.responses['401'], `${o.id} must document 401`);
    }
  }
});

test('every state-changing operation except sign-in requires the CSRF token header', () => {
  for (const o of operations()) {
    const csrf = headerParams(o.op).find((p) => p.name === 'X-CSRF-Token');
    if (o.method === 'get' || isSignIn(o)) {
      assert.equal(csrf, undefined, `${o.id} must not require a CSRF token`);
    } else {
      assert.ok(csrf?.required, `${o.id} must require X-CSRF-Token`);
      assert.ok(o.op.responses['403'], `${o.id} must document 403 for a missing CSRF token`);
    }
  }
});

test('message submission requires an Idempotency-Key, like the public API', () => {
  const create = spec.paths['/portal/v1/messages'].post;
  const key = headerParams(create).find((p) => p.name === 'Idempotency-Key');
  assert.ok(key?.required);
  const publicKey = resolve.call(null, publicSpec.components.parameters.IdempotencyKey);
  assert.equal(key.schema.pattern, publicKey.schema.pattern);
});

test('message operations declare the permission they require', () => {
  for (const o of operations()) {
    if (o.path === '/portal/v1/session') continue;
    const want = o.method === 'get' ? 'messages:read' : 'messages:send';
    assert.equal(o.op['x-required-permission'], want, o.id);
  }
});

test('no request lets the browser choose a tenant', () => {
  for (const o of operations()) {
    for (const p of (o.op.parameters ?? []).map(resolve)) {
      assert.doesNotMatch(p.name, /tenant|account|organization/i, `${o.id} parameter ${p.name}`);
    }
    const schema = resolve(o.op.requestBody?.content?.['application/json']?.schema);
    if (schema) {
      assert.equal(schema.additionalProperties, false, `${o.id} request body must be closed`);
      for (const name of Object.keys(schema.properties)) {
        assert.doesNotMatch(name, /tenant|account|organization/i, `${o.id} body field ${name}`);
      }
    }
  }
});

test('statuses, categories and encodings match the public API', () => {
  for (const name of ['MessageStatus', 'StatusCategory']) {
    assert.deepEqual(spec.components.schemas[name].enum, publicSpec.components.schemas[name].enum, name);
  }
  assert.deepEqual(
    spec.components.schemas.Encoding.enum,
    publicSpec.components.schemas.Message.properties.encoding.enum,
  );
});

test('responses never contain the message text or provider identifiers', () => {
  for (const name of ['Message', 'MessageDetail', 'StatusChange']) {
    const text = JSON.stringify(spec.components.schemas[name]);
    for (const forbidden of ['"body"', 'provider_message', 'provider_id', 'password']) {
      assert.ok(!text.includes(forbidden), `${name} must not contain ${forbidden}`);
    }
  }
});
