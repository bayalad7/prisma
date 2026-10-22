// Capturas de docs/administracion/primeros-pasos.md (WebP): recorre el primer uso en un puente
// recién instalado, de crear la contraseña a autorizar la primera aplicación.
//
// Necesita Node 18+, Playwright (npm i playwright), un puente SIN configurar abierto desde el propio
// equipo y una impresora de red; sirve la de prueba: `prisma dev impresora-virtual --puerto 9100`.
//
//   PRISMA_URL=https://127.0.0.1:8443 PRISMA_IMPRESORA=127.0.0.1:9100 node capturar.mjs
//
// Crea en ese puente la contraseña «una frase larga de prueba», la impresora «caja» y la aplicación
// «Punto de venta»: úsalo solo con un puente de prueba. Las imágenes se escriben junto a este archivo
// (o en PRISMA_SALIDA).
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const B = process.env.PRISMA_URL || 'https://127.0.0.1:8443';
const IMPRESORA = process.env.PRISMA_IMPRESORA || '127.0.0.1:9100';
const AQUI = process.env.PRISMA_SALIDA || path.dirname(fileURLToPath(import.meta.url));
const CLAVE = 'una frase larga de prueba';
const espera = (ms) => new Promise((r) => setTimeout(r, ms));

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
const listo = async (page) => {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(500);
};
// Lo que se ve: el diálogo abierto, si hay uno; si no, la pantalla.
const dialogo = async (page) => ((await page.locator('dialog[open]').count()) ? page.locator('dialog[open]').last() : page);

const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 }, ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: 'light' });
const page = await ctx.newPage();
await page.goto(`${B}/admin/`); await espera(1000);
if (!(await page.locator('#nueva').count())) throw new Error('El puente ya tiene contraseña: usa uno recién instalado.');
await guardar(page, page.locator('.acceso .tarjeta'), '01-crear-contrasena');
await page.fill('#nueva', CLAVE);
await page.fill('#repetir', CLAVE);
await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
await listo(page);
await guardar(page, page, '02-inicio');

// Impresora: el asistente de tres pasos.
await page.goto(`${B}/admin/impresoras`); await listo(page);
await page.getByRole('button', { name: 'Agregar impresora' }).first().click(); await espera(500);
await guardar(page, await dialogo(page), '03-tipo');
await page.locator('label.opcion', { hasText: 'De tickets de 80 mm' }).click();
await page.getByRole('button', { name: 'Continuar' }).click(); await espera(2500);
await page.locator('summary', { hasText: '¿No aparece?' }).click();
await page.locator('#as-destino').fill(IMPRESORA);
await page.locator('#as-nombre').fill('Caja');
await page.locator('#as-destino').focus();
await espera(300);
await guardar(page, await dialogo(page), '04-conexion');
await page.getByRole('button', { name: 'Guardar y probar' }).click();
await page.getByText('¿Salió el ticket?').waitFor({ timeout: 20000 }); await espera(500);
await guardar(page, await dialogo(page), '05-prueba');
await page.getByRole('button', { name: 'Sí, salió' }).click(); await espera(300);
await page.getByRole('button', { name: 'Sí, se leen bien' }).click(); await espera(800);
await guardar(page, await dialogo(page), '06-listo');
await page.keyboard.press('Escape'); await espera(400);

// Aplicación: autorizar la página del punto de venta.
await page.goto(`${B}/admin/aplicaciones`); await listo(page);
await page.getByRole('button', { name: 'Agregar aplicación' }).first().click(); await espera(400);
await page.locator('#cli-nombre').fill('Punto de venta');
await page.locator('#cli-origen').fill('https://pos.mitienda.mx');
await page.getByRole('button', { name: 'Agregar', exact: true }).click(); await espera(300);
await guardar(page, await dialogo(page), '07-aplicacion');
await page.getByRole('button', { name: 'Crear aplicación' }).click(); await espera(1200);
await guardar(page, await dialogo(page), '08-llave');
await page.getByRole('button', { name: 'Ya la guardé' }).click(); await listo(page);
await page.locator('.tarjeta', { hasText: 'Punto de venta' }).first().getByRole('button', { name: 'Código para integrar' }).click();
await page.waitForSelector('dialog[open]'); await espera(400);
await guardar(page, await dialogo(page), '09-codigo');
await page.keyboard.press('Escape'); await espera(300);

await page.goto(`${B}/admin/`); await listo(page);
await guardar(page, page, '10-inicio-listo');
await browser.close();
