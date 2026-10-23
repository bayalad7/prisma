// Pruebas de punta a punta del panel (Fase 7 de 0.2.0): los flujos a)–g) en escritorio y en celular,
// cada uno contra su propio puente recién instalado (puente.mjs), con axe en cada pantalla.
// Uso: cd e2e && npm ci && npx playwright install chromium && npm test
import { defineConfig, devices } from '@playwright/test';

// Las pruebas también llaman a la API como lo haría una aplicación (fetch de Node y el SDK); el
// puente de prueba usa un certificado autofirmado.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

const PUERTOS = { escritorio: 18601, celular: 18602 };

export default defineConfig({
  testDir: '.',
  testMatch: /.*\.spec\.mjs/,
  timeout: 120_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 2,
  retries: 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: { ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: 'light', trace: 'retain-on-failure', screenshot: 'only-on-failure' },
  projects: [
    { name: 'escritorio', use: { viewport: { width: 1366, height: 768 }, baseURL: `https://127.0.0.1:${PUERTOS.escritorio}` } },
    { name: 'celular', use: { ...devices['Pixel 7'], viewport: { width: 390, height: 844 }, baseURL: `https://127.0.0.1:${PUERTOS.celular}` } },
  ],
  webServer: Object.values(PUERTOS).map((puerto) => ({
    command: `node puente.mjs ${puerto}`,
    url: `https://127.0.0.1:${puerto}/v1/salud`,
    ignoreHTTPSErrors: true,
    timeout: 60_000,
    reuseExistingServer: false,
  })),
});
