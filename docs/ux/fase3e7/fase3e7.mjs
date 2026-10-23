// Fase 3, entrega 7: tarjetas de Impresoras, Imágenes y pantallas de acceso. Uso: node fase3e7.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
const AQUI = path.dirname(fileURLToPath(import.meta.url));
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [] };
const comprobar = (nombre, ok, detalle = '') => informe.comprobaciones.push(`${ok ? 'ok   ' : 'FALLA'} ${nombre} ${detalle}`);
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};
async function revisar(page, clave) {
  const a = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = a.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((x) => x.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(400);
}
async function contexto(browser, vp) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/40[13]|Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  return { ctx, page };
}
async function abrir(browser, vp, ruta) {
  const { ctx, page } = await contexto(browser, vp);
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await listo(page);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const sesionFalsa = (page, datos) => page.route('**/admin/api/sesion', (r) => r.fulfill({ json: datos }));

const browser = await chromium.launch();
for (const vp of ['escritorio', 'celular']) {
  // Pantallas de acceso.
  const { ctx, page } = await contexto(browser, vp);
  await page.goto(`${B}/admin/`); await espera(800);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click(); await espera(200);
  comprobar(`${vp}: sin contraseña, error junto al campo`, await page.getByText('Escribe la contraseña.').isVisible());
  await page.fill('#contrasena', 'no-es-esta');
  const ver = page.getByRole('button', { name: 'Mostrar la contraseña' });
  await ver.click();
  comprobar(`${vp}: «mostrar» deja ver la contraseña`, await page.locator('#contrasena').getAttribute('type') === 'text'
    && await page.getByRole('button', { name: 'Ocultar la contraseña' }).getAttribute('aria-pressed') === 'true');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click(); await espera(1200);
  comprobar(`${vp}: contraseña equivocada explicada`, await page.getByText('La contraseña no es correcta.').isVisible());
  await page.getByText('¿Olvidaste la contraseña?').click(); await espera(200);
  await cap(page, `01-entrar-${vp}`, false);
  await revisar(page, `entrar-${vp}`);

  await sesionFalsa(page, { configurado: false, autenticado: false, local: false });
  await page.route('**/admin/api/configuracion-inicial', (r) => r.fulfill({ status: 403, json: { codigo: 'CODIGO_INVALIDO', mensaje: 'El código de configuración no es correcto.', sugerencia: 'Revisa el código que mostró la terminal.' } }));
  await page.reload(); await espera(800);
  await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click(); await espera(200);
  comprobar(`${vp}: configuración inicial marca código y contraseña`, await page.locator('#codigo[aria-invalid="true"], #nueva[aria-invalid="true"]').count() === 2);
  await page.locator('#codigo').fill('0000-AAAA');
  await page.locator('#nueva').fill('una frase larga de prueba');
  await page.locator('#repetir').fill('una frase larga de prueba');
  await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click(); await espera(600);
  comprobar(`${vp}: código equivocado marcado en su campo`, await page.locator('#codigo').getAttribute('aria-invalid') === 'true');
  await cap(page, `02-configuracion-inicial-${vp}`);
  await revisar(page, `configuracion-${vp}`);

  await page.unroute('**/admin/api/sesion');
  await page.route('**/admin/api/sesion', (r) => r.abort());
  await page.reload(); await espera(800);
  comprobar(`${vp}: sin conexión ofrece reintentar`, await page.getByRole('button', { name: 'Volver a intentar' }).isVisible());
  await cap(page, `03-sin-conexion-${vp}`, false);
  await revisar(page, `sin-conexion-${vp}`);
  await page.unroute('**/admin/api/sesion');
  await page.getByRole('button', { name: 'Volver a intentar' }).click(); await espera(800);
  comprobar(`${vp}: al reintentar vuelve a pedir la contraseña`, await page.locator('#contrasena').isVisible());
  await ctx.close();
}
for (const vp of ['escritorio', 'celular']) {
  const { ctx, page } = await abrir(browser, vp, '/admin/impresoras');
  await cap(page, `04-impresoras-${vp}`);
  await revisar(page, `impresoras-${vp}`);
  comprobar(`${vp}: tarjetas con insignias, sin pastillas`, await page.locator('.insignia').count() > 0 && await page.locator('.pastilla').count() === 0);
  if (vp === 'escritorio') {
    const tarjeta = page.locator('.lista > .tarjeta').filter({ has: page.getByRole('button', { name: 'Desactivar' }) }).first();
    const alias = await tarjeta.locator('h2, h3').first().innerText();
    await tarjeta.getByRole('button', { name: 'Desactivar' }).click(); await page.waitForSelector('dialog'); await espera(300);
    comprobar('desactivar pide confirmación con consecuencias', await page.locator('dialog li').count() >= 2);
    await cap(page, '05-desactivar-escritorio', false);
    await page.locator('dialog').getByRole('button', { name: 'Desactivar' }).click(); await listo(page); await espera(600);
    const t2 = page.locator('.lista > .tarjeta').filter({ hasText: alias }).first();
    comprobar('desactivada: insignia y «Activar» como acción principal', await t2.getByText('Desactivada').isVisible() && await t2.getByRole('button', { name: 'Activar' }).isVisible());
    await cap(page, '06-desactivada-escritorio');
    await t2.getByRole('button', { name: 'Activar' }).click(); await listo(page); await espera(600);
    comprobar('activar la devuelve', await page.locator('.lista > .tarjeta').filter({ hasText: alias }).first().getByRole('button', { name: 'Desactivar' }).isVisible());
  }
  await ctx.close();
}
for (const vp of ['escritorio', 'celular']) {
  const { ctx, page } = await abrir(browser, vp, '/admin/imagenes');
  await page.getByRole('button', { name: 'Guardar imagen' }).click(); await espera(200);
  comprobar(`${vp}: guardar sin imagen lo explica junto a la zona`, await page.getByText('Elige primero una imagen PNG.').isVisible());
  await page.setInputFiles('#rec-archivo', path.join(AQUI, 'logo-prueba.png')); await espera(1200);
  await page.locator('#rec-nombre').fill('Logo Malo'); await page.locator('#rec-nombre').blur(); await espera(200);
  comprobar(`${vp}: nombre inválido marcado en su campo`, await page.locator('#rec-nombre').getAttribute('aria-invalid') === 'true');
  await cap(page, `07-imagenes-${vp}`);
  await revisar(page, `imagenes-${vp}`);
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
