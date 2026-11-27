// Generates src/lib/api/schema.gen.ts, the TypeScript types of the client portal API, from the
// approved contract contracts/openapi/portal-api-v1.yaml with openapi-typescript (owner decision G3-6,
// decision log §20). Types only: no runtime code is generated.
//
//   pnpm --filter @sms/client-portal generate:api         write the file
//   pnpm --filter @sms/client-portal generate:api --check fail if the committed file is stale
//
// Never edit schema.gen.ts by hand: change the contract (through its own review) and regenerate.
// Plain Node.js (no shell), so the same command works on Linux, macOS and Windows.
import { readFileSync, writeFileSync } from 'node:fs';
import openapiTS, { astToString } from 'openapi-typescript';

const contract = new URL('../../../../contracts/openapi/portal-api-v1.yaml', import.meta.url);
const output = new URL('../src/lib/api/schema.gen.ts', import.meta.url);

const header = `// Code generated from contracts/openapi/portal-api-v1.yaml by openapi-typescript. DO NOT EDIT.
// Regenerate with: pnpm --filter @sms/client-portal generate:api
/* eslint-disable */
`;

const ast = await openapiTS(contract, { alphabetize: true });
const generated = header + astToString(ast);

if (process.argv.includes('--check')) {
  let committed = '';
  try {
    committed = readFileSync(output, 'utf8');
  } catch {
    // reported below
  }
  if (committed !== generated) {
    console.error(
      'src/lib/api/schema.gen.ts is stale or was edited by hand. Run "pnpm --filter @sms/client-portal generate:api" and commit the result.',
    );
    process.exit(1);
  }
  console.log('src/lib/api/schema.gen.ts is up to date.');
} else {
  writeFileSync(output, generated);
  console.log('wrote src/lib/api/schema.gen.ts');
}
