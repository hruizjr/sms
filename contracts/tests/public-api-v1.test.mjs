// Contract tests for contracts/openapi/public-api-v1.yaml.
//
// Redocly (`pnpm run lint`) checks OpenAPI validity, style rules, and that every example matches its
// schema. These tests check the project-specific rules that a generic linter cannot: API conventions
// (AGENTS.md §9), the idempotency contract (docs/architecture/api-idempotency.md), the message status
// semantics (docs/architecture/messaging-reliability.md), gateway independence, and that the
// specification and its README agree. They read the bundled specification, so run `pnpm run bundle`
// first (`pnpm run check` does both). Standard library only (node:test).

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const root = new URL('..', import.meta.url);
const bundlePath = fileURLToPath(new URL('build/public-api-v1.json', root));
if (!existsSync(bundlePath)) {
  throw new Error(`${bundlePath} not found: run "pnpm run bundle" first (or "pnpm run check")`);
}
const spec = JSON.parse(readFileSync(bundlePath, 'utf8'));
const readme = readFileSync(fileURLToPath(new URL('openapi/README.md', root)), 'utf8');

const METHODS = ['get', 'put', 'post', 'delete', 'patch', 'head', 'options', 'trace'];

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
    const parts = n.$ref.replace(/^#\//, '').split('/').map((p) => p.replace(/~1/g, '/').replace(/~0/g, '~'));
    n = parts.reduce((acc, key) => acc[key], spec);
  }
  return n;
}

// Visits every schema object reachable from the components and paths (after bundling, $refs are local).
function walkSchemas(schema, visit, seen = new Set()) {
  const s = resolve(schema);
  if (!s || typeof s !== 'object' || seen.has(s)) return;
  seen.add(s);
  visit(s);
  for (const key of ['items', 'not', 'additionalProperties']) {
    if (s[key] && typeof s[key] === 'object') walkSchemas(s[key], visit, seen);
  }
  for (const key of ['oneOf', 'anyOf', 'allOf']) {
    for (const sub of s[key] ?? []) walkSchemas(sub, visit, seen);
  }
  for (const sub of Object.values(s.properties ?? {})) walkSchemas(sub, visit, seen);
}

function allText(node, out = []) {
  if (typeof node === 'string') out.push(node);
  else if (Array.isArray(node)) node.forEach((n) => allText(n, out));
  else if (node && typeof node === 'object') {
    for (const [k, v] of Object.entries(node)) {
      out.push(k);
      allText(v, out);
    }
  }
  return out;
}

const createOp = () => spec.paths['/v1/messages'].post;

// ---------------------------------------------------------------------------------------------
// Conventions (AGENTS.md §9)

test('every path is versioned under /v1', () => {
  for (const path of Object.keys(spec.paths)) assert.match(path, /^\/v1\//, path);
});

test('every operation declares authentication, a required permission, and an operationId', () => {
  assert.deepEqual(spec.security, [{ api_key: [] }], 'global security must require the API key');
  for (const { op, id } of operations()) {
    assert.ok(op.operationId, `${id}: operationId`);
    assert.notDeepEqual(op.security, [], `${id}: must not disable authentication`);
    assert.match(op['x-required-permission'] ?? '', /^[a-z_]+:[a-z_]+$/, `${id}: x-required-permission`);
    assert.match(op.description, /Required permission:/, `${id}: description states the permission`);
    assert.match(op.description, /Rate limits:/, `${id}: description states rate limiting`);
    assert.ok(op.responses['401'] && op.responses['403'] && op.responses['429'], `${id}: 401, 403, 429`);
  }
});

test('the API key is sent in the Authorization header, never in the URL', () => {
  const schemes = spec.components.securitySchemes;
  assert.deepEqual(Object.keys(schemes), ['api_key']);
  assert.equal(schemes.api_key.type, 'http');
  assert.equal(schemes.api_key.scheme, 'bearer');
  for (const { op, id } of operations()) {
    for (const p of op.parameters ?? []) {
      const param = resolve(p);
      if (param.in === 'header') continue; // Idempotency-Key is not a credential
      assert.doesNotMatch(param.name, /key|token|secret|password|auth/i, `${id}: credential-like ${param.in} parameter ${param.name}`);
    }
  }
});

test('every error response is RFC 9457 problem details and carries X-Request-ID', () => {
  for (const { op, id } of operations()) {
    for (const [code, response] of Object.entries(op.responses)) {
      const r = resolve(response);
      assert.ok(r.headers?.['X-Request-ID'], `${id} ${code}: X-Request-ID header`);
      if (Number(code) >= 400) {
        const media = Object.keys(r.content ?? {});
        assert.deepEqual(media, ['application/problem+json'], `${id} ${code}: media type`);
        assert.equal(resolve(r.content['application/problem+json'].schema), spec.components.schemas.Problem, `${id} ${code}: Problem schema`);
      }
    }
  }
});

test('every problem example uses about:blank or a documented problem type, with a matching status', () => {
  const documented = new Set([...readme.matchAll(/^\| `([a-z-]+)` \| (\d{3}) \|/gm)].map((m) => `${m[1]}:${m[2]}`));
  assert.ok(documented.size >= 10, 'README problem type table not found');
  for (const { op, id } of operations()) {
    for (const [code, response] of Object.entries(op.responses)) {
      if (Number(code) < 400) continue;
      const content = resolve(response).content['application/problem+json'];
      const examples = content.examples ? Object.values(content.examples).map((e) => e.value) : [content.example];
      assert.ok(examples.length > 0 && examples.every(Boolean), `${id} ${code}: problem example`);
      for (const ex of examples) {
        assert.equal(ex.status, Number(code), `${id} ${code}: example status`);
        if (ex.type === 'about:blank') continue;
        const slug = ex.type.replace(/^https:\/\/api\.example\.invalid\/problems\//, '');
        assert.ok(documented.has(`${slug}:${code}`), `${id} ${code}: problem type ${ex.type} is not in the README table`);
      }
    }
  }
});

test('all JSON property names are snake_case', () => {
  const bad = [];
  for (const schema of Object.values(spec.components.schemas)) {
    walkSchemas(schema, (s) => {
      for (const name of Object.keys(s.properties ?? {})) if (!/^[a-z][a-z0-9_]*$/.test(name)) bad.push(name);
    });
  }
  for (const { op } of operations()) {
    for (const p of op.parameters ?? []) {
      const param = resolve(p);
      if (param.in === 'query' || param.in === 'path') {
        if (!/^[a-z][a-z0-9_]*$/.test(param.name)) bad.push(param.name);
      }
    }
  }
  assert.deepEqual(bad, []);
});

test('timestamps are date-time strings named *_at (or *_after / *_before filters)', () => {
  for (const schema of Object.values(spec.components.schemas)) {
    walkSchemas(schema, (s) => {
      for (const [name, prop] of Object.entries(s.properties ?? {})) {
        const p = resolve(prop);
        if (p.format === 'date-time') assert.match(name, /_at$/, name);
        if (name.endsWith('_at')) assert.equal(p.format, 'date-time', name);
      }
    });
  }
});

test('request bodies reject unknown fields', () => {
  for (const { op, id } of operations()) {
    if (!op.requestBody) continue;
    const schema = resolve(op.requestBody.content['application/json'].schema);
    assert.equal(schema.additionalProperties, false, id);
  }
});

test('collections use cursor pagination', () => {
  for (const { op, id, method } of operations()) {
    if (method !== 'get') continue;
    const ok = resolve(op.responses['200']).content['application/json'].schema;
    const schema = resolve(ok);
    if (schema.properties?.data?.type !== 'array') continue;
    const names = (op.parameters ?? []).map((p) => resolve(p).name);
    assert.ok(names.includes('limit') && names.includes('cursor'), `${id}: limit and cursor`);
    assert.ok(schema.required.includes('next_cursor'), `${id}: next_cursor`);
  }
});

// ---------------------------------------------------------------------------------------------
// Phone numbers and identifiers

test('the E.164 pattern accepts valid numbers and rejects others', () => {
  const re = new RegExp(spec.components.schemas.PhoneNumber.pattern);
  for (const ok of ['+639000000001', '+6390000000', '+1234567']) assert.ok(re.test(ok), ok);
  for (const bad of ['09000000001', '639000000001', '+63 900 000 0001', '+63-900-000-0001', '+0639000000001', '+1234567890123456', '', '+']) {
    assert.ok(!re.test(bad), bad);
  }
});

test('the Idempotency-Key pattern accepts tokens and quoted strings, and rejects malformed values', () => {
  const re = new RegExp(spec.components.parameters.IdempotencyKey.schema.pattern);
  for (const ok of ['0c9d4bf1-3b52-4f6e-9a57-3f0b6f2f9b1e', '"0c9d4bf1-3b52-4f6e-9a57-3f0b6f2f9b1e"', 'a', 'x'.repeat(255)]) {
    assert.ok(re.test(ok), ok);
  }
  for (const bad of ['', '"', '""', '"abc', 'abc"', 'a b', 'a\\b', 'x'.repeat(256), 'é']) assert.ok(!re.test(bad), JSON.stringify(bad));
});

test('no example uses a phone number outside the synthetic range', () => {
  const numbers = allText(spec).join('\n').match(/\+\d{7,15}/g) ?? [];
  assert.ok(numbers.length > 0);
  for (const n of numbers) assert.match(n, /^\+6390000000\d\d$/, `${n} is not a synthetic example number`);
});

// ---------------------------------------------------------------------------------------------
// Message submission and idempotency (api-idempotency.md, messaging-reliability.md §3)

test('message submission requires an Idempotency-Key header', () => {
  const params = createOp().parameters.map(resolve);
  const key = params.find((p) => p.name === 'Idempotency-Key');
  assert.ok(key, 'Idempotency-Key parameter');
  assert.equal(key.in, 'header');
  assert.equal(key.required, true);
});

test('message submission returns 202 Accepted (asynchronous), never 200 or 201', () => {
  const codes = Object.keys(createOp().responses);
  assert.ok(codes.includes('202'));
  assert.ok(!codes.includes('200') && !codes.includes('201'));
  const ok = resolve(createOp().responses['202']);
  assert.ok(ok.headers['Idempotent-Replayed'], 'replay header');
  assert.ok(ok.headers.Location, 'Location header');
  const example = Object.values(ok.content['application/json'].examples)[0].value;
  assert.equal(example.status, 'accepted');
  assert.equal(example.status_category, 'in_progress');
});

test('message submission documents every idempotency outcome', () => {
  const r = createOp().responses;
  assert.ok(r['400'], 'missing/invalid key');
  assert.ok(r['409'] && resolve(r['409']).headers['Retry-After'], '409 in progress with Retry-After');
  assert.ok(r['422'], '422 key reused');
  const text = JSON.stringify(createOp()) + JSON.stringify(spec.components);
  for (const type of ['idempotency-key-missing', 'idempotency-key-invalid', 'idempotency-request-in-progress', 'idempotency-key-reused']) {
    assert.ok(text.includes(type), type);
  }
});

test('the message text is never returned in a response', () => {
  assert.ok(spec.components.schemas.MessageCreateRequest.properties.body);
  const responseSchemas = new Set();
  for (const { op } of operations()) {
    for (const [code, response] of Object.entries(op.responses)) {
      if (Number(code) >= 300) continue;
      for (const media of Object.values(resolve(response).content ?? {})) {
        walkSchemas(media.schema, (s) => responseSchemas.add(s));
      }
    }
  }
  for (const s of responseSchemas) assert.ok(!('body' in (s.properties ?? {})), 'response must not contain body');
});

test('no network-side identifiers or raw provider codes are exposed', () => {
  for (const schema of Object.values(spec.components.schemas)) {
    walkSchemas(schema, (s) => {
      for (const name of Object.keys(s.properties ?? {})) {
        assert.doesNotMatch(name, /provider|telco|network|smsc|carrier|operator/, `property ${name}`);
      }
    });
  }
});

// ---------------------------------------------------------------------------------------------
// Status model (messaging-reliability.md §3; README status reference)

const statusEnum = spec.components.schemas.MessageStatus.enum;
const categoryEnum = spec.components.schemas.StatusCategory.enum;
const readmeStatuses = new Map(
  [...readme.matchAll(/^\| `([a-z_]+)` \| `(in_progress|uncertain|final)` \| ([^|]+) \|/gm)].map((m) => [m[1], { category: m[2], internal: m[3] }]),
);

test('status values and categories in the specification match the README status reference', () => {
  assert.deepEqual([...readmeStatuses.keys()].sort(), [...statusEnum].sort());
  assert.deepEqual(categoryEnum, ['in_progress', 'uncertain', 'final']);
  const categoryText = spec.components.schemas.StatusCategory.description;
  for (const [status, { category }] of readmeStatuses) {
    const line = categoryText.split('\n').find((l) => l.includes(`\`${category}\``));
    assert.ok(line?.includes(`\`${status}\``), `StatusCategory description lists ${status} under ${category}`);
  }
});

test('acceptance and network acknowledgment are distinct statuses', () => {
  assert.ok(statusEnum.includes('accepted'));
  assert.ok(statusEnum.includes('provider_accepted'));
  assert.ok(!statusEnum.includes('sent'), '"sent" is ambiguous between hand-off and acknowledgment');
});

test('uncertain outcomes are explicit statuses and are never categorized as final or failed', () => {
  for (const s of ['submission_uncertain', 'unknown']) {
    assert.ok(statusEnum.includes(s), s);
    assert.equal(readmeStatuses.get(s).category, 'uncertain', s);
  }
  const d = spec.components.schemas.MessageStatus.description;
  assert.match(d, /`submission_uncertain`[^]*may already be delivered/i);
  assert.match(d, /`unknown`[^]*not\*\* proof of non-delivery/i);
});

test('every internal lifecycle state (ADR-0021) maps to exactly one customer status', () => {
  const internal = ['ACCEPTED', 'QUEUED', 'SUBMITTING', 'SUBMIT_UNCERTAIN', 'SUBMITTED', 'REJECTED', 'FAILED', 'DELIVERED', 'UNDELIVERED', 'UNKNOWN'];
  const seen = new Map();
  for (const [status, { internal: cell }] of readmeStatuses) {
    for (const state of cell.match(/[A-Z_]{4,}/g) ?? []) {
      assert.ok(!seen.has(state), `${state} mapped twice (${seen.get(state)}, ${status})`);
      seen.set(state, status);
    }
  }
  assert.deepEqual([...seen.keys()].sort(), [...internal].sort());
});

test('every documented status reason code is valid and attached to an existing status', () => {
  const desc = spec.components.schemas.StatusReason.properties.code.description;
  const codes = [...desc.matchAll(/`([a-z_]+)`/g)].map((m) => m[1]).filter((c) => !statusEnum.includes(c));
  assert.ok(codes.length >= 8, 'reason codes listed');
  const pattern = new RegExp(spec.components.schemas.StatusReason.properties.code.pattern);
  for (const c of codes) assert.ok(pattern.test(c), c);
  for (const m of desc.matchAll(/\(with `([a-z_]+)`\)/g)) assert.ok(statusEnum.includes(m[1]), m[1]);
});

// ---------------------------------------------------------------------------------------------
// Gateway independence and reliability wording

test('the contract contains no gateway, protocol, or infrastructure specifics', () => {
  const text = allText(spec).join('\n');
  const forbidden = /\b(smpp|submit_sm|deliver_sm|esme|smsc|jasmin|rabbitmq|amqp|redis|kafka|redpanda|ton|npi|bind|dlr|smart|globe|pldt|dito)\b/i;
  const hit = text.split('\n').find((l) => forbidden.test(l));
  assert.equal(hit, undefined, `forbidden term in: ${hit}`);
});

test('exactly-once delivery is only ever mentioned to deny it', () => {
  const text = allText(spec).join('\n').replace(/\s+/g, ' ');
  const sentences = text.split(/(?<=[.;:])\s/).filter((s) => /exactly[- ]once/i.test(s));
  assert.ok(sentences.length > 0, 'the contract must state that exactly-once delivery is not promised');
  for (const s of sentences) assert.match(s, /\b(not|cannot|never|no)\b/i, s);
});

test('a 202 is documented as acceptance only, not submission or delivery', () => {
  const d = (spec.info.description + createOp().description).replace(/\s+/g, ' ');
  assert.match(d, /does \*\*not\*\* mean that the message was handed to a provider, that a provider or the recipient.s mobile network accepted it, or that it reached the recipient.s handset/);
  assert.match(d, /Acceptance is not submission and not delivery/);
});

// ---------------------------------------------------------------------------------------------
// Provider semantics (owner review of PR #7): "provider" is our immediate upstream SMS provider;
// its acknowledgment is not necessarily acceptance by the recipient's mobile network, and delivery
// outcomes are provider-reported, not independently verified.

const flat = (text) => text.replace(/\s+/g, ' ');
const statusDescription = flat(spec.components.schemas.MessageStatus.description);
const statusItem = (status) => statusDescription.split(' - ').find((item) => item.startsWith(`\`${status}\``)) ?? '';
const readmeRow = (status) => readme.split('\n').find((l) => l.startsWith(`| \`${status}\` |`)) ?? '';

test('provider_accepted means the immediate upstream provider acknowledged, not the destination network', () => {
  for (const [where, text] of [['MessageStatus', statusItem('provider_accepted')], ['README', readmeRow('provider_accepted')]]) {
    assert.match(text, /immediate upstream SMS provider acknowledged/, `${where}: who acknowledged`);
    assert.match(text, /not\*{0,2} necessarily .*recipient's mobile network/i, `${where}: not necessarily the mobile network`);
  }
  assert.match(flat(spec.info.description), /\*\*Provider\*\* means the platform's immediate upstream SMS provider/);
  assert.match(flat(readme), /\*\*Provider\*\* means the platform's \*\*immediate upstream SMS provider\*\*/);
});

test('delivered and undelivered are provider-reported outcomes, not independently verified facts', () => {
  for (const status of ['delivered', 'undelivered']) {
    for (const [where, text] of [['MessageStatus', statusItem(status)], ['README', readmeRow(status)]]) {
      assert.match(text, /provider-reported delivery outcome/i, `${where} ${status}: provider-reported`);
      assert.match(text, /not (an )?independently verified/i, `${where} ${status}: not independently verified`);
    }
  }
  assert.match(flat(spec.components.schemas.StatusCategory.description), /Provider-reported finals .* not an independently verified fact/);
});

test('acknowledgments and reports are never attributed to "the network"', () => {
  const sentences = allText(spec).map(flat).join(' ').split(/(?<=[.;])\s/);
  const hit = sentences.find((s) => /\b(a|any|the) network\b/i.test(s));
  assert.equal(hit, undefined, `ambiguous network wording: ${hit}`);
});

// ---------------------------------------------------------------------------------------------
// X-Request-ID (owner review of PR #7): the contract must match the Group 1 middleware
// (backend/internal/middleware/requestid.go), which reuses a valid client value and otherwise
// generates one, without rejecting the request.

const middlewareSource = readFileSync(fileURLToPath(new URL('../backend/internal/middleware/requestid.go', root)), 'utf8');

test('the Group 1 middleware still behaves as the contract describes', () => {
  assert.match(middlewareSource, /const RequestIDHeader = "X-Request-ID"/);
  assert.match(middlewareSource, /const maxRequestIDLength = 128/);
  assert.match(middlewareSource, /case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '\.', c == '_', c == '-':/);
  assert.match(middlewareSource, /if !validRequestID\(id\) \{\s*id = generate\(\)/, 'invalid values are replaced, not rejected');
});

test('every operation documents the optional X-Request-ID request header', () => {
  const param = spec.components.parameters.RequestId;
  assert.equal(param.name, 'X-Request-ID');
  assert.equal(param.in, 'header');
  assert.equal(param.required, false);
  // Invalid values are ignored, not rejected, so the parameter schema must not constrain them.
  assert.deepEqual(param.schema, { type: 'string' });
  assert.match(flat(param.description), /1-128 characters, all from `\[A-Za-z0-9\._-\]`/);
  assert.match(flat(param.description), /\*\*ignored, not rejected\*\*/);
  for (const { op, id } of operations()) {
    assert.ok((op.parameters ?? []).some((p) => p.$ref === '#/components/parameters/RequestId'), `${id}: X-Request-ID parameter`);
  }
});

test('the X-Request-ID response header matches what the middleware can return', () => {
  const header = spec.components.headers.XRequestID;
  assert.equal(header.schema.pattern, '^[A-Za-z0-9._-]{1,128}$');
  const re = new RegExp(header.schema.pattern);
  for (const ok of ['a', 'client-req-7f3a2c', '0123456789abcdef0123456789abcdef', 'x'.repeat(128)]) assert.ok(re.test(ok), ok);
  for (const bad of ['', 'x'.repeat(129), 'a b', 'a/b', 'é']) assert.ok(!re.test(bad), JSON.stringify(bad));
  assert.doesNotMatch(flat(header.description), /echo/i, 'no unconditional echo promise');
});

// ---------------------------------------------------------------------------------------------
// Idempotency behaviour (owner check before merge) and owner decisions of 2026-11-25.

const keyDescription = flat(spec.components.parameters.IdempotencyKey.description);

test('same account, operation, key, and body after completion: the original 202 is replayed', () => {
  assert.match(keyDescription, /Scope: your account \+ this operation/);
  assert.match(keyDescription, /3\. \*\*Identical body, original completed:\*\* the original `202` response and body are replayed, with `Idempotent-Replayed: true`/);
  const accepted = flat(resolve(createOp().responses['202']).description);
  assert.match(accepted, /On an idempotent replay the same status code and body are returned with `Idempotent-Replayed: true`/);
  assert.match(flat(createOp().description), /does not create a second message or a second charge/);
});

test('same key with a different body: documented 422 idempotency-key-reused, nothing processed', () => {
  assert.match(keyDescription, /1\. \*\*Different normalized body:\*\* `422` \(`idempotency-key-reused`\), whether the original request is still being processed or has completed\. Nothing is processed\./);
  const r = resolve(createOp().responses['422']);
  const ex = r.content['application/problem+json'].examples.idempotencyKeyReused.value;
  assert.equal(ex.status, 422);
  assert.equal(ex.type, 'https://api.example.invalid/problems/idempotency-key-reused');
  assert.match(readme, /^\| `idempotency-key-reused` \| 422 \| Same key, different normalized body — even while the original is still being processed \|$/m);
});

test('retention is seven days and the behaviour after expiry is explicit', () => {
  assert.match(keyDescription, /\*\*Retention: seven days\.\*\* A recorded key is kept for seven days \(168 hours\) after the original request was accepted/);
  assert.match(keyDescription, /processed as a \*\*new\*\* request: it is not compared with the earlier body \(no `422`\), no earlier response is replayed, and it can create a second message, a second charge, and a second SMS/);
  assert.match(keyDescription, /Do not retry with an old key after seven days/);
  assert.doesNotMatch(keyDescription, /not yet decided|Provisional/i);
  assert.match(readme, /\*\*Retention: seven days\*\* \(D-094d\)/);
  assert.match(readme, /\*\*After retention expires\*\* the key is forgotten/);
});

test('every owner decision listed in the contract README is recorded in the register and the workbook', () => {
  const register = readFileSync(fileURLToPath(new URL('../docs/requirements/discovery-register.md', root)), 'utf8');
  const workbook = readFileSync(fileURLToPath(new URL('../docs/discovery/owner-discovery-workbook.md', root)), 'utf8');
  const section = readme.slice(readme.indexOf('## 7. Owner decisions'), readme.indexOf('**Still provisional or open**'));
  const ids = [...new Set([...section.matchAll(/^\| (D-\d{3})/gm)].map((m) => m[1]))];
  assert.deepEqual(ids, ['D-094', 'D-096', 'D-099', 'D-143', 'D-144', 'D-145', 'D-146', 'D-147']);
  for (const id of ids) {
    const regRow = register.split('\n').find((l) => l.startsWith(`| ${id} |`)) ?? '';
    const wbRow = workbook.split('\n').find((l) => l.startsWith(`| ${id} |`)) ?? '';
    assert.match(regRow, /Owner decision 2026-11-25/, `${id}: discovery register`);
    assert.match(wbRow, /2026-11-25/, `${id}: owner discovery workbook`);
  }
});

// ---------------------------------------------------------------------------------------------
// Final owner decisions (2026-11-25): idempotency outcome order, body-only comparison, ADR-0018 and
// ADR-0021 acceptance, and bulk sending deferred through both channels.

const docText = (path) => readFileSync(fileURLToPath(new URL(path, root)), 'utf8');

test('idempotency outcomes are evaluated in order: different body 422, then 409 in progress, then replay', () => {
  const steps = [...keyDescription.matchAll(/(\d)\. \*\*([^*]+)\*\*/g)].map((m) => `${m[1]} ${m[2]}`);
  assert.deepEqual(steps, ['1 Different normalized body:', '2 Identical body, original still being processed:', '3 Identical body, original completed:']);
  assert.match(keyDescription, /2\. \*\*Identical body, original still being processed:\*\* `409` \(`idempotency-request-in-progress`\) with `Retry-After`/);
  const inProgress = flat(spec.components.responses.IdempotencyRequestInProgress.description);
  assert.match(inProgress, /same `Idempotency-Key` \*\*and an identical normalized body\*\* is still being processed/);
  assert.match(inProgress, /different body with the same key returns `422 idempotency-key-reused` instead, even while the original is still being processed/);
  const rows = readme.split('\n').filter((l) => /^ {2}\| [123] \|/.test(l)).map((l) => l.trim());
  assert.deepEqual(rows.map((r) => r.split('|')[3].trim().slice(0, 4)), ['`422', '`409', 'Orig']);
  const idem = flat(docText('../docs/architecture/api-idempotency.md'));
  assert.match(idem, /a different fingerprint returns `422` even while the original is still in progress/);
});

test('only the normalized request body is compared; X-Request-ID and tracing/transport headers are not', () => {
  assert.match(keyDescription, /\*\*Comparison: the normalized request body only\.\*\*/);
  assert.match(keyDescription, /`X-Request-ID` and other tracing or transport headers are \*\*not\*\* part of the comparison/);
  assert.match(flat(createOp().description), /Only the normalized request body is compared, never `X-Request-ID` or other tracing or transport headers/);
  assert.match(flat(readme), /`X-Request-ID` and other tracing or transport headers are \*\*not\*\* part of the comparison/);
  assert.match(flat(docText('../docs/architecture/api-idempotency.md')), /`X-Request-ID` and other tracing or transport headers are \*\*not\*\* part of the comparison/);
});

test('ADR-0018 is accepted, ADR-0021 and ADR-0012 are accepted with a limited scope, ADR-0010 is accepted for control-plane tables only, and no other ADR status changed', () => {
  const log = docText('../docs/engineering/decision-log.md');
  const index = log.slice(log.indexOf('## 4. Index'), log.indexOf('## 5. Review summary'));
  const accepted = [...index.matchAll(/^\| (\d{4}) \| [^|]+ \| \*\*Accepted/gm)].map((m) => m[1]);
  // ADR-0010 accepted by the owner on 2026-11-26 (decision log §12.1, RLS-1 to RLS-5).
  // ADR-0012 accepted by the owner on 2026-11-26 with a limited scope (decision log §15, EX-3 simulator-only).
  assert.deepEqual(accepted, ['0002', '0003', '0004', '0005', '0010', '0012', '0013', '0014', '0015', '0016', '0017', '0018', '0021']);
  assert.match(index, /^\| 0012 \| [^|]+ \| \*\*Accepted — limited scope\*\* \(owner, 2026-11-26\): option B limited to the provisional interface, capability declarations, adapter abstraction and simulator; no SMPP library or real adapter \|$/m);
  const adr12 = flat(log.slice(log.indexOf('### ADR-0012'), log.indexOf('### ADR-0013')));
  assert.match(adr12, /\*\*Status:\*\* \*\*Accepted — limited scope\*\* \(by owner, 2026-11-26\)/);
  assert.match(adr12, /does \*\*not\*\* cover, and these remain \*\*separate decisions\*\*: the \*\*SMPP client library\*\* selection/);
  assert.match(adr12, /\*\*any real telco adapter\*\*/);
  assert.match(index, /^\| 0010 \| [^|]+ \| \*\*Accepted\*\* \(owner, 2026-11-26\): option B for tenant-owned control-plane tables; high-volume data-plane tables \*\*deferred\*\* \(RLS-5\) \|$/m);
  const adr10 = flat(log.slice(log.indexOf('### ADR-0010'), log.indexOf('### ADR-0011')));
  assert.match(adr10, /\*\*Status:\*\* \*\*Accepted\*\* \(by owner, 2026-11-26\) — \*\*option B for tenant-owned control-plane tables\*\*; high-volume data-plane tables excluded and deferred/);
  assert.match(adr10, /\*\*Deferred \(RLS-5\):\*\* this decision does \*\*not\*\* extend to high-volume data-plane \(message\) tables/);
  assert.match(index, /^\| 0021 \| [^|]+ \| \*\*Accepted — limited scope\*\*/m);
  const adr21 = flat(log.slice(log.indexOf('### ADR-0021'), log.indexOf('## 7. Owner decisions')));
  assert.match(adr21, /\*\*Status:\*\* \*\*Accepted — limited scope\*\*/);
  assert.match(adr21, /does \*\*not\*\* settle, and these remain \*\*separate open decisions\*\*/);
  assert.match(adr21, /billing treatment of `unknown`\*\*/);
  assert.match(adr21, /\*\*webhook behaviour\*\* \(D-099, webhook part\)/);
  const adr18 = flat(log.slice(log.indexOf('### ADR-0018'), log.indexOf('### ADR-0019')));
  assert.match(adr18, /\*\*Status:\*\* \*\*Accepted\*\* \(by owner, 2026-11-25\)/);
  assert.match(adr18, /whether the original request is still processing or has completed/);
});

test('bulk sending is deferred through both channels and absent from the contract', () => {
  for (const path of Object.keys(spec.paths)) assert.doesNotMatch(path, /bulk|batch/i, path);
  assert.equal(spec.components.schemas.MessageCreateRequest.properties.to.$ref ?? spec.components.schemas.MessageCreateRequest.properties.to.type, '#/components/schemas/PhoneNumber', 'one recipient per request');
  const brief = flat(docText('../docs/product/product-brief.md'));
  assert.match(brief, /Single-message send via the client portal and the public API \(one recipient per message\)/);
  assert.match(brief, /\*\*Bulk sending\*\* \(many recipients per send\) through the public API and through the client portal, including uploaded recipient files — deferred to a later release/);
  const worksheet = docText('../docs/discovery/mvp-scope-worksheet.md');
  assert.match(worksheet, /^\| MVP-08 \|.*\*\*Deferred to a later release\*\*/m);
  assert.match(worksheet, /^\| MVP-09 \|.*\*\*MVP: single-message sending only\*\*.*\*\*deferred to a later release\*\*/m);
  const register = docText('../docs/requirements/discovery-register.md');
  assert.match(register.split('\n').find((l) => l.startsWith('| D-089 |')), /bulk sending through both channels \(MVP-08, MVP-09\) is deferred to a later release/);
});

test('MVP portal: exactly one recipient per send operation (MVP-09)', () => {
  const worksheet = docText('../docs/discovery/mvp-scope-worksheet.md');
  const row = worksheet.split('\n').find((l) => l.startsWith('| MVP-09 |'));
  assert.match(row, /exactly one recipient per send operation; no pasted recipient lists, recipient-file uploads, contact-group sends, or multi-recipient selection/);
  assert.match(flat(worksheet), /one portal send operation accepts \*\*exactly one recipient\*\*/);
  const brief = flat(docText('../docs/product/product-brief.md'));
  assert.match(brief, /one send operation accepts \*\*exactly one recipient\*\*: no pasted recipient lists, recipient-file uploads, contact-group sends, or multi-recipient selection/);
  assert.match(brief, /several separate single-recipient messages, subject to rate limits and abuse controls/);
});

test('MVP-13 campaign approval is deferred while essential abuse controls stay in the MVP', () => {
  const worksheet = docText('../docs/discovery/mvp-scope-worksheet.md');
  const rows = Object.fromEntries(worksheet.split('\n').filter((l) => /^\| MVP-\d\d \|/.test(l)).map((l) => [l.slice(2, 8), l]));
  assert.match(rows['MVP-13'], /\*\*Deferrable\*\* \| \*\*Deferred beyond the MVP\*\*/);
  for (const id of ['MVP-01', 'MVP-05', 'MVP-29', 'MVP-30']) assert.match(rows[id], /\| Essential \| \*\*Essential\*\* — .*\(owner, 2026-11-25\) \|$/, id);
  const register = docText('../docs/requirements/discovery-register.md');
  for (const id of ['D-025', 'D-087']) {
    const line = register.split('\n').find((l) => l.startsWith(`| ${id} |`));
    assert.match(line, /campaign approval workflow is \*\*deferred from the MVP\*\*/, id);
    assert.match(line, /thresholds, first-campaign .*policies, and operational turnaround remain open for a later release/, id);
  }
  const brief = flat(docText('../docs/product/product-brief.md'));
  assert.match(brief, /\*\*Campaign approval workflows\*\* .* deferred beyond the MVP/);
  assert.match(brief, /Abuse controls: tenant\/API-key velocity limits, abuse alerts, and an operational kill switch/);
});

test('MVP-11 saved contacts/groups and MVP-12 campaign scheduling are deferred (owner, 2026-11-26)', () => {
  const worksheet = docText('../docs/discovery/mvp-scope-worksheet.md');
  const row = (id) => worksheet.split('\n').find((l) => l.startsWith(`| ${id} |`));
  assert.match(row('MVP-11'), /\| \*\*Deferrable\*\* — saved contacts and contact groups deferred from the MVP; the MVP portal takes a manually entered mobile number per single-recipient message \(owner, 2026-11-26\) \|$/);
  assert.match(row('MVP-12'), /\| \*\*Deferrable\*\* — campaign scheduling deferred .*a separate scheduled-single-message feature is not ruled out \(owner, 2026-11-26\) \|$/);
  const brief = flat(docText('../docs/product/product-brief.md'));
  assert.match(brief, /\*\*Saved contacts and contact groups\*\* .* deferred from the MVP .* Saved contacts may be introduced independently/);
  assert.match(brief, /\*\*Campaign scheduling\*\* — deferred .* A separate scheduled-single-message feature is not ruled out/);
  const register = docText('../docs/requirements/discovery-register.md');
  assert.match(register.split('\n').find((l) => l.startsWith('| D-089 |')), /Owner decision 2026-11-26/);
  // D-078 (reviewer and operational ownership) stays open: MVP-11/12 did not decide it, and the EX-1 scope
  // approval of 2026-11-26 only names the internal roles (MVP-04).
  const d078 = register.split('\n').find((l) => l.startsWith('| D-078 |'));
  assert.match(d078, /Item remains open\./);
  assert.match(d078, /Staff numbers and turnaround remain open\. \|$/);
});
