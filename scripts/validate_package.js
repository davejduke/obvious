const path = require('path');
const pkg = require(path.join(process.cwd(), 'frontend/package.json'));
const deps = {...pkg.dependencies, ...pkg.devDependencies};

const required = ['react', 'react-dom', 'next', 'typescript', 'tailwindcss'];
required.forEach(d => {
  if (!deps[d]) throw new Error('Missing dep: ' + d);
  console.log('OK dep: ' + d + ' ' + deps[d]);
});

const reactVer = deps['react'].replace(/[\^~]/g, '');
if (!reactVer.startsWith('19')) throw new Error('React 19 required, got: ' + reactVer);

const nextVer = deps['next'].replace(/[\^~]/g, '');
if (!nextVer.startsWith('15')) throw new Error('Next.js 15 required, got: ' + nextVer);

console.log('React 19 + Next.js 15: OK');
