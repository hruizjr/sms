// Contract tests for contracts/openapi/portal-api-v1.yaml (Phase 4 Group 3; owner decisions G3-1 to
// G3-6; Group 6: audit log, API key last use, sign-in rate limit). Redocly checks validity and style; these tests check the security rules of the portal API
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

test('every operation except the session declares the permission it requires', () => {
  for (const o of operations()) {
    if (o.path === '/portal/v1/session') continue;
    const want = o.path.startsWith('/portal/v1/api-keys')
      ? 'api_keys:manage'
      : o.path.startsWith('/portal/v1/audit-events')
        ? 'audit_log:read'
        : o.method === 'get'
        ? 'messages:read'
        : 'messages:send';
    assert.equal(o.op['x-required-permission'], want, o.id);
  }
});

test('API key scopes are exactly the public API permissions', () => {
  const publicPermissions = new Set();
  for (const item of Object.values(publicSpec.paths)) {
    for (const method of METHODS) {
      const permission = item[method]?.['x-required-permission'];
      if (permission) publicPermissions.add(permission);
    }
  }
  assert.deepEqual([...spec.components.schemas.ApiKeyScope.enum].sort(), [...publicPermissions].sort());
});

test('API key metadata never contains a secret; only the create response returns the key, uncached', () => {
  const properties = Object.keys(spec.components.schemas.ApiKey.properties);
  for (const name of properties) {
    assert.doesNotMatch(name, /secret|hash|^key$|token|tenant/, `ApiKey property ${name}`);
  }
  assert.equal(spec.components.schemas.ApiKeyList.properties.data.items.$ref, '#/components/schemas/ApiKey');
  const keyBearing = Object.entries(spec.components.schemas)
    .filter(([, schema]) => schema.properties?.key !== undefined)
    .map(([name]) => name);
  assert.deepEqual(keyBearing, ['ApiKeyCreated']);
  const created = spec.paths['/portal/v1/api-keys'].post.responses['201'];
  assert.equal(created.content['application/json'].schema.$ref, '#/components/schemas/ApiKeyCreated');
  assert.deepEqual(created.headers['Cache-Control'].schema.enum, ['no-store']);
});

test('creating an API key requires re-authentication with the current password', () => {
  const request = spec.components.schemas.ApiKeyCreateRequest;
  assert.ok(request.required.includes('current_password'));
  assert.equal(request.properties.current_password.writeOnly, true);
  assert.equal(request.properties.current_password.format, 'password');
  assert.ok(spec.paths['/portal/v1/api-keys'].post.responses['403']);
  for (const name of ['ApiKey', 'ApiKeyCreated', 'Session']) {
    assert.ok(!JSON.stringify(spec.components.schemas[name]).includes('password'), `${name} must not contain a password`);
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

test('the audit log is read-only and never contains secrets, credentials or message text', () => {
  const item = spec.paths['/portal/v1/audit-events'];
  assert.deepEqual(Object.keys(item).filter((k) => METHODS.includes(k)), ['get']);
  assert.equal(item.get['x-required-permission'], 'audit_log:read');
  assert.ok(item.get.responses['403']);
  const names = ['AuditEvent', 'AuditParty', 'AuditDetails'].flatMap((name) =>
    Object.keys(spec.components.schemas[name].properties),
  );
  for (const name of names) {
    assert.doesNotMatch(name, /password|secret|hash|token|authorization|cookie|^key$|body|text|tenant/i, `audit field ${name}`);
  }
  assert.equal(spec.components.schemas.AuditDetails.additionalProperties, false);
  // Only owners hold audit_log:read, as for API key management.
  assert.ok(spec.components.schemas.Permission.enum.includes('audit_log:read'));
});

test('sign-in declares the per-source rate limit with Retry-After', () => {
  const tooMany = resolve(spec.paths['/portal/v1/session'].post.responses['429']);
  assert.ok(tooMany, 'POST /portal/v1/session must declare 429');
  assert.ok(resolve(tooMany.headers['Retry-After']), '429 must declare Retry-After');
  assert.match(tooMany.content['application/problem+json'].example.type, /too-many-sign-in-attempts$/);
});

test('API keys report when they were last used, nullable', () => {
  const key = spec.components.schemas.ApiKey;
  assert.ok(key.required.includes('last_used_at'));
  assert.deepEqual(
    key.properties.last_used_at.oneOf.map((s) => s.type),
    ['string', 'null'],
  );
});
