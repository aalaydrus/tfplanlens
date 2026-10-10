// Full-page screenshots of the smoke-test HTML reports, loaded from file://
// (so this also proves the report works offline).
//
//   node scripts/screenshot.mjs [smoke-dir] [screenshots-dir]
//
// SCREENSHOT_SCENARIOS=simple,complex limits it to those scenarios (default:
// all). Screenshots are committed, so keep the list short.
//
// Needs Playwright: npm i --no-save playwright && npx playwright install chromium
import { chromium } from 'playwright';
import { readdirSync, mkdirSync, existsSync } from 'node:fs';
import { resolve, join } from 'node:path';

const smokeDir = resolve(process.argv[2] ?? 'docs/smoke');
const outDir = resolve(process.argv[3] ?? 'docs/screenshots');
mkdirSync(outDir, { recursive: true });
const only = process.env.SCREENSHOT_SCENARIOS ? process.env.SCREENSHOT_SCENARIOS.split(',') : null;

const viewports = [
  { name: 'desktop', width: 1280, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

const browser = await chromium.launch();
let failed = false;
for (const scenario of readdirSync(smokeDir, { withFileTypes: true }).filter((d) => d.isDirectory())) {
  if (only && !only.includes(scenario.name)) continue;
  const report = join(smokeDir, scenario.name, 'report.html');
  if (!existsSync(report)) continue;
  for (const vp of viewports) {
    const page = await browser.newPage({ viewport: { width: vp.width, height: vp.height } });
    const problems = [];
    page.on('pageerror', (e) => problems.push(`page error: ${e}`));
    page.on('console', (m) => m.type() === 'error' && problems.push(`console: ${m.text()}`));
    page.on('request', (r) => !/^(file|data):/.test(r.url()) && problems.push(`external request: ${r.url()}`));
    await page.goto('file://' + report);
    const overflow = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
    }));
    const file = join(outDir, `${scenario.name}-${vp.name}.png`);
    await page.screenshot({ path: file, fullPage: true });
    const horizontal = overflow.scrollWidth > overflow.innerWidth;
    console.log(`${file}: page ${overflow.scrollWidth}px in ${overflow.innerWidth}px viewport` +
      (horizontal ? ' (HORIZONTAL OVERFLOW)' : '') + (problems.length ? ` problems: ${problems.join('; ')}` : ''));
    if (problems.length) failed = true;

    // Detail drawers hold the attribute diffs, so capture the interesting
    // ones too (desktop only): the sensitive marker and the decoded
    // user_data diff of the complex scenario.
    if (vp.name === 'desktop') {
      for (const [slug, address] of [['creds', 'terraform_data.creds'], ['user_data', 'terraform_data.user_data']]) {
        const row = page.locator('tr.row', { hasText: address }).first();
        if ((await row.count()) === 0) continue;
        await row.click();
        const drawer = join(outDir, `${scenario.name}-drawer-${slug}.png`);
        await page.screenshot({ path: drawer });
        console.log(`${drawer}: drawer for ${address}`);
        await page.keyboard.press('Escape');
      }
    }
    await page.close();
  }
}
await browser.close();
process.exit(failed ? 1 : 0);
