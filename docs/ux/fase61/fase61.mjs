// Fase 6.1: impresora virtual de principio a fin en el panel real, con axe y capturas.
// Uso: node fase61.mjs <carpeta-salida> <llave pk_ de una app de nivel 2 con origen https://dev.ejemplo.mx>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

const [SALIDA, LLAVE] = process.argv.slice(2);
const B = 'https://127.0.0.1:18443';
const P = `${B}/admin/`;
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], api: [] };

async function imprimirDesdeApp(archivo, referencia) {
  const doc = JSON.parse(fs.readFileSync(path.join('C:/prisma/ejemplos/validos', archivo), 'utf8'));
  Object.assign(doc, { id: crypto.randomUUID(), impresora: 'prueba', referencia });
  delete doc.firma;
  const r = await fetch(`${B}/v1/imprimir`, { method: 'POST', headers: { 'Content-Type': 'application/json', Origin: 'https://dev.ejemplo.mx', Authorization: `Bearer ${LLAVE}` }, body: JSON.stringify(doc) });
  const j = await r.json();
  informe.api.push({ archivo, estado: r.status, codigo: j.codigo, trabajo: j.trabajo_id, advertencias: (j.advertencias || []).map((a) => a.codigo) });
  return j;
}
async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((n) => n.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function ir(page, seccion) {
  await page.goto(`${P}#${seccion}`);
  await page.evaluate(() => dispatchEvent(new HashChangeEvent('hashchange')));
  await page.waitForSelector('#cargando', { state: 'detached' }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(400);
}
const tarjeta = (page, alias) => page.locator('article').filter({ has: page.locator('.alias', { hasText: new RegExp(`^${alias}$`) }) });

const browser = await chromium.launch();
for (const [vp, op] of Object.entries({
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
})) {
  const ctx = await browser.newContext({ ...op, ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(P); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); await espera(1500); }

  if (vp === 'escritorio') {
    // Asistente: ofrece la impresora virtual y la crea.
    await ir(page, 'impresoras');
    await page.getByRole('button', { name: 'Agregar impresora' }).first().click(); await espera(2500);
    await page.screenshot({ path: path.join(SALIDA, `01-asistente-ofrece-virtual-${vp}.png`) });
    await revisar(page, `asistente-${vp}`);
    await page.getByRole('button', { name: 'Crear impresora virtual de prueba' }).click(); await espera(1500);
    await page.waitForSelector('#cargando', { state: 'detached' }).catch(() => {});
    await page.screenshot({ path: path.join(SALIDA, `02-impresoras-con-virtual-${vp}.png`), fullPage: true });
    // Diagnóstico en la virtual: imagen del ticket.
    await tarjeta(page, 'prueba').getByRole('button', { name: 'Ticket de diagnóstico' }).click();
    await page.waitForSelector('dialog'); await espera(600);
    await page.screenshot({ path: path.join(SALIDA, `03-diagnostico-virtual-${vp}.png`) });
    await revisar(page, `diagnostico-${vp}`);
    await page.keyboard.press('Escape'); await espera(1200);
    // Una aplicación real imprime por la API v1.
    await imprimirDesdeApp('venta-simple.json', 'VTA-2001');
    await imprimirDesdeApp('comanda-cocina.json', 'PED-0310');
  }

  await ir(page, 'impresoras');
  await tarjeta(page, 'prueba').getByRole('button', { name: /Ver tickets/ }).click();
  await page.waitForSelector('dialog'); await espera(1200);
  await page.screenshot({ path: path.join(SALIDA, `04-rollo-${vp}.png`) });
  await revisar(page, `rollo-${vp}`);
  if (vp === 'escritorio') {
    await page.locator('dialog details summary').first().click(); await espera(800);
    await page.locator('dialog details').first().screenshot({ path: path.join(SALIDA, `05-traza-${vp}.png`) });
  }
  await page.keyboard.press('Escape'); await espera(300);

  if (vp === 'escritorio') {
    // Simulación: fuera de línea, desde el formulario.
    await tarjeta(page, 'prueba').getByRole('button', { name: 'Editar' }).click(); await espera(500);
    await page.selectOption('#simulacion', 'lenta'); await espera(200);
    await page.screenshot({ path: path.join(SALIDA, `06-simulacion-${vp}.png`), fullPage: true });
    await revisar(page, `formulario-${vp}`);
    await page.selectOption('#simulacion', 'fuera_de_linea');
    await page.getByRole('button', { name: 'Guardar cambios' }).click(); await espera(1500);
    await imprimirDesdeApp('venta-simple.json', 'VTA-2002');
    await ir(page, 'impresoras');
    await tarjeta(page, 'prueba').screenshot({ path: path.join(SALIDA, `07-tarjeta-fuera-de-linea-${vp}.png`) });
  }
  await ir(page, 'trabajos');
  await page.screenshot({ path: path.join(SALIDA, `08-trabajos-${vp}.png`), fullPage: vp === 'celular' });
  await revisar(page, `trabajos-${vp}`);
  await ir(page, 'inicio');
  await page.screenshot({ path: path.join(SALIDA, `09-inicio-${vp}.png`), fullPage: true });
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log('api:', JSON.stringify(informe.api));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
