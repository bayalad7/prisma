// Capturas de docs/administracion/probar-sin-impresora.md, en escritorio y celular (WebP).
//
// Necesita Node 18+, Playwright (npm i playwright) y un puente de prueba con datos: una impresora
// virtual «prueba» en línea y una aplicación de nivel 2 con todas las impresoras. Uso:
//
//   PRISMA_URL=https://127.0.0.1:18443 PRISMA_CONTRASENA=… PRISMA_ORIGEN=https://app.example \
//   PRISMA_LLAVE=pk_… NODE_TLS_REJECT_UNAUTHORIZED=0 node capturar.mjs
//
// La llave y el origen son de la aplicación de prueba: se usan para imprimir un ticket por /v1 y
// verlo llegar al rollo. Las imágenes se escriben junto a este archivo (o en PRISMA_SALIDA).
import { chromium } from 'playwright';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const B = process.env.PRISMA_URL || 'https://127.0.0.1:8443';
const CLAVE = process.env.PRISMA_CONTRASENA;
const ORIGEN = process.env.PRISMA_ORIGEN;
const LLAVE = process.env.PRISMA_LLAVE;
const VIRTUAL = process.env.PRISMA_VIRTUAL || 'prueba';
const AQUI = process.env.PRISMA_SALIDA || path.dirname(fileURLToPath(import.meta.url));
if (!CLAVE || !ORIGEN || !LLAVE) {
  console.error('Faltan PRISMA_CONTRASENA, PRISMA_ORIGEN o PRISMA_LLAVE.');
  process.exit(2);
}
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 820 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

async function guardar(page, objetivo, nombre) {
  const png = await objetivo.screenshot({ type: 'png' });
  const webp = await page.evaluate(async (b64) => {
    const img = new Image();
    img.src = `data:image/png;base64,${b64}`;
    await img.decode();
    const c = document.createElement('canvas');
    c.width = img.naturalWidth; c.height = img.naturalHeight;
    c.getContext('2d').drawImage(img, 0, 0);
    return c.toDataURL('image/webp', 0.85).split(',')[1];
  }, png.toString('base64'));
  fs.writeFileSync(path.join(AQUI, `${nombre}.webp`), Buffer.from(webp, 'base64'));
  console.log('ok', nombre);
}
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: 'light' });
  const page = await ctx.newPage();
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) {
    await page.fill('#contrasena', CLAVE);
    await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  }
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(800);
  return { ctx, page };
}
async function imprimirDesdeApp(referencia) {
  const r = await fetch(`${B}/v1/imprimir`, { method: 'POST',
    headers: { 'Content-Type': 'application/json', Origin: ORIGEN, Authorization: `Bearer ${LLAVE}` },
    body: JSON.stringify({ version: '1', id: crypto.randomUUID(), impresora: VIRTUAL, referencia, operaciones: [
      { tipo: 'texto', valor: 'CAFETERÍA EL PORTAL', alineacion: 'centro', negrita: true, tamano: [2, 2] },
      { tipo: 'separador' },
      { tipo: 'par', izquierda: '1 x Café americano', derecha: '$38.00' },
      { tipo: 'par', izquierda: '2 x Pan de elote', derecha: '$70.00' },
      { tipo: 'par', izquierda: 'TOTAL', derecha: '$108.00', negrita: true },
      { tipo: 'qr', valor: 'https://example.com/ticket/1', alineacion: 'centro' },
      { tipo: 'cortar' }] }) });
  if (!r.ok) throw new Error(`/v1/imprimir respondió ${r.status}: ${await r.text()}`);
}

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/impresoras');
  await guardar(page, page.locator('.lista > .tarjeta').filter({ hasText: VIRTUAL }).first(), 'tarjeta-virtual');
  await page.goto(`${B}/admin/pruebas?impresora=${VIRTUAL}`); await page.waitForSelector('.previa-prueba img'); await espera(600);
  await guardar(page, page, 'probar-impresion-escritorio');
  await page.goto(`${B}/admin/pruebas/rollo/${VIRTUAL}`); await espera(1500);
  await imprimirDesdeApp('VTA-0001');
  await page.waitForSelector('.ticket-rollo', { timeout: 8000 }); await espera(1200);
  await guardar(page, page, 'rollo-escritorio');
  await page.goto(`${B}/admin/pruebas/permisos`); await espera(1500);
  await guardar(page, page, 'permisos-escritorio');
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', `/admin/pruebas/rollo/${VIRTUAL}`);
  await page.waitForSelector('.ticket-rollo', { timeout: 8000 }).catch(() => {}); await espera(800);
  await guardar(page, page.locator('.ticket-rollo').first(), 'rollo-celular');
  await page.goto(`${B}/admin/pruebas?impresora=${VIRTUAL}`); await page.waitForSelector('.previa-prueba img'); await espera(600);
  await guardar(page, page.locator('.tarjeta-previa'), 'vista-previa-celular');
  await ctx.close();
}
await browser.close();
