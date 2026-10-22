// Fase 3, entrega 3: asistente de impresoras y prueba de acentos.
// A) impresora de red (simulada en 127.0.0.1:9100) con prueba y acentos; B) virtual en celular;
// C) errores de validación. Uso: node fase3e3.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [] };
const comprobar = (nombre, ok, detalle = '') => informe.comprobaciones.push(`${ok ? 'ok   ' : 'FALLA'} ${nombre} ${detalle}`);
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};
const sufijo = Date.now() % 10000;

async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((n) => n.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function abrir(browser, vp) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}/admin/impresoras`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await page.waitForSelector('.vista .esqueleto', { state: 'detached' }).catch(() => {}); await espera(400);
  await page.getByRole('button', { name: 'Agregar impresora' }).first().click(); await espera(500);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });

const browser = await chromium.launch();

// A) Impresora de red en escritorio.
{
  const { ctx, page } = await abrir(browser, 'escritorio');
  await cap(page, '01-paso1-tipo-escritorio');
  await revisar(page, 'paso1-escritorio');
  await page.getByRole('button', { name: 'Continuar' }).click(); await espera(2500);
  await cap(page, '02-paso2-conexion-escritorio');
  await revisar(page, 'paso2-escritorio');
  comprobar('marca las impresoras que no son de tickets', await page.getByText('No parece de tickets').count() > 0);
  await page.locator('summary', { hasText: '¿No aparece?' }).click();
  await page.locator('#as-destino').fill('127.0.0.1:9100');
  await page.locator('#as-nombre').fill(`Mostrador ${sufijo}`);
  comprobar('propone el alias a partir del nombre', await page.locator('#as-alias').inputValue() === `mostrador-${sufijo}`, await page.locator('#as-alias').inputValue());
  await page.getByRole('button', { name: 'Probar conexión' }).click(); await espera(1500);
  comprobar('probar conexión dice que respondió', await page.getByText('La impresora respondió: está lista.').isVisible());
  await cap(page, '03-paso2-probado-escritorio');
  await page.getByRole('button', { name: 'Guardar y probar' }).click();
  await page.getByText('¿Salió el ticket?').waitFor({ timeout: 20000 }); await espera(500);
  await cap(page, '04-paso3-salio-escritorio');
  await revisar(page, 'paso3-escritorio');
  await page.getByRole('button', { name: 'Sí, salió' }).click(); await espera(200);
  await page.getByRole('button', { name: 'No, salen símbolos raros' }).click();
  await page.waitForSelector('dialog'); await espera(500);
  await cap(page, '05-acentos-escritorio', false);
  await revisar(page, 'acentos-escritorio');
  await page.locator('dialog').getByRole('button', { name: 'En ninguno se leen bien' }).click(); await espera(200);
  comprobar('«En ninguno» explica qué hacer sin cerrar', await page.locator('dialog').getByText('Tu impresora usa otra tabla').isVisible());
  await page.locator('dialog label.opcion').nth(1).click();
  await page.locator('dialog').getByRole('button', { name: 'Usar este renglón' }).click(); await espera(800);
  comprobar('termina con «Listo»', await page.getByText(/Listo: «mostrador-/).isVisible());
  await cap(page, '06-listo-escritorio');
  await page.goto(`${B}/admin/impresoras`); await espera(1200);
  const perfil = await page.locator('article').filter({ hasText: `mostrador-${sufijo}` }).textContent();
  comprobar('la página de códigos elegida quedó guardada (CP850)', perfil.includes('CP850'));
  await ctx.close();
}
// B) Virtual en celular.
{
  const { ctx, page } = await abrir(browser, 'celular');
  await page.locator('label.opcion', { hasText: 'Virtual, para probar' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click(); await espera(400);
  await cap(page, '07-virtual-paso2-celular');
  await revisar(page, 'virtual-paso2-celular');
  await page.getByRole('button', { name: 'Crear y probar' }).click();
  await page.getByText(/Listo: «prueba/).waitFor({ timeout: 20000 }); await espera(400);
  await cap(page, '08-virtual-listo-celular');
  await revisar(page, 'virtual-listo-celular');
  await ctx.close();
}
// C) Validación en escritorio.
{
  const { ctx, page } = await abrir(browser, 'escritorio');
  await page.getByRole('button', { name: 'Continuar' }).click(); await espera(800);
  await page.locator('#as-alias').fill('Caja Principal');
  await page.getByRole('button', { name: 'Guardar y probar' }).click(); await espera(300);
  comprobar('error en línea del nombre', await page.getByText('Escribe un nombre para la impresora.').isVisible());
  comprobar('error en línea del alias', await page.getByText(/Usa solo minúsculas/).isVisible());
  comprobar('pide elegir la conexión', await page.getByText('Elige una impresora de la lista o escribe su dirección.').isVisible());
  comprobar('el campo con error queda marcado', await page.locator('#as-nombre').getAttribute('aria-invalid') === 'true');
  await cap(page, '09-errores-escritorio');
  await revisar(page, 'errores-escritorio');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
