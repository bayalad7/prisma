// Pruebas de punta a punta del panel (Fase 7 de 0.2.0): los flujos a)–g) en escritorio y en celular,
// cada uno contra su propio puente recién instalado (puente.mjs), y después todas las pantallas en
// claro y oscuro y la navegación con teclado, con axe en cada pantalla.
// Uso: cd e2e && npm ci && npx playwright install chromium && npm test
import { defineConfig, devices } from '@playwright/test';

// Las pruebas también llaman a la API como lo haría una aplicación (fetch de Node y el SDK); el
// puente de prueba usa un certificado autofirmado.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

const PUERTOS = { escritorio: 18601, celular: 18602 };
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 }, baseURL: `https://127.0.0.1:${PUERTOS.escritorio}` },
  celular: { ...devices['Pixel 7'], viewport: { width: 390, height: 844 }, baseURL: `https://127.0.0.1:${PUERTOS.celular}` },
};

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
  // Primero los flujos (dejan el puente con datos); después, en el mismo puente, todas las pantallas y
  // el teclado.
  projects: [
    { name: 'escritorio', testMatch: /flujos\.spec\.mjs/, use: VISTAS.escritorio },
    { name: 'celular', testMatch: /flujos\.spec\.mjs/, use: VISTAS.celular },
    { name: 'escritorio-pantallas', testMatch: /(pantallas|teclado)\.spec\.mjs/, dependencies: ['escritorio'], use: VISTAS.escritorio },
    { name: 'celular-pantallas', testMatch: /pantallas\.spec\.mjs/, dependencies: ['celular'], use: VISTAS.celular },
    // Firefox real contra la autoridad local del puente de escritorio (ADR 0021).
    { name: 'certificado-firefox', testMatch: /certificado-firefox\.spec\.mjs/, use: VISTAS.escritorio },
    // Con su propio puente (lo reinicia): de recién instalado a imprimir, y el reinicio.
    { name: 'usuario-nuevo', testMatch: /usuario-nuevo\.spec\.mjs/, use: { ...VISTAS.escritorio, baseURL: 'https://127.0.0.1:18603' } },
  ],
  webServer: Object.values(PUERTOS).map((puerto) => ({
    command: `node puente.mjs ${puerto}`,
    url: `https://127.0.0.1:${puerto}/v1/salud`,
    ignoreHTTPSErrors: true,
    timeout: 60_000,
    reuseExistingServer: false,
  })),
});
