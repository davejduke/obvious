const fs = require('fs');
const path = require('path');
const src = fs.readFileSync(path.join(process.cwd(), 'shared/types/typescript/index.ts'), 'utf8');

const types = ['Persona', 'Organization', 'Control', 'Engagement', 'Finding',
               'NIS2Article', 'NIS2ComplianceScore', 'ApiResponse', 'PaginatedResponse'];
types.forEach(t => {
  if (!src.includes(t)) throw new Error('Missing type: ' + t);
  console.log('OK type: ' + t);
});

const personas = ['internal_auditor', 'cae', 'audit_committee', 'auditee_ciso',
                  'cosourced_provider', 'ptwg_member', 'beta_tester'];
personas.forEach(p => {
  if (!src.includes(p)) throw new Error('Missing persona: ' + p);
});
console.log('All 7 personas present');

const articles = ['21a','21b','21c','21d','21e','21f','21g','21h','21i','21j'];
articles.forEach(a => {
  if (!src.includes('"' + a + '"')) throw new Error('Missing NIS2 article: ' + a);
});
console.log('All 10 NIS2 articles present');
