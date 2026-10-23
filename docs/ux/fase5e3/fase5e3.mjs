// Fase 5, entrega 3: galería, diagramas de cada pantalla y de cada error, versiones sencillas. Uso: node fase5e3.mjs <carpeta-salida>
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
async function revisar(page, clave) {
  const a = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = a.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((x) => x.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(300);
}
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: 'light' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await listo(page);
  return { ctx, page };
}
async function irA(page, ruta) {
  await page.evaluate((r) => { history.pushState(null, '', r); dispatchEvent(new PopStateEvent('popstate')); }, ruta);
  await espera(700); await listo(page);
}
const cap = (page, nombre, completa = false) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const titulos = (page) => page.locator('.tarjeta-diagrama strong').allInnerTexts();

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ayuda');
  await page.getByRole('link', { name: 'Ver los diagramas' }).click(); await listo(page); await espera(500);
  const negocio = await titulos(page);
  comprobar('la galería empieza por «Para el negocio», con las versiones sencillas primero',
    page.url().endsWith('/admin/ayuda/diagramas') && negocio[0] === 'Cómo funciona PRISMA' && negocio.includes('El viaje de un ticket')
    && !negocio.includes('Componentes y dependencias'), negocio.slice(0, 4).join(' | '));
  await page.waitForFunction(() => [...document.querySelectorAll('.tarjeta-diagrama .solo-claro')].slice(0, 3).every((i) => i.complete && i.naturalWidth > 0), null, { timeout: 8000 }).catch(() => {});
  await cap(page, '01-galeria-escritorio', true);
  await revisar(page, 'galeria-claro');
  await page.getByRole('tab', { name: 'Técnicos' }).click(); await espera(300);
  const tecnicos = await titulos(page);
  comprobar('«Técnicos» muestra los de quien programa y desarrolla', tecnicos.includes('Componentes y dependencias') && !tecnicos.includes('Cómo funciona PRISMA'), `${tecnicos.length}`);
  await page.locator('#ga-buscar').fill('autorizacion'); await espera(300);
  comprobar('la búsqueda no distingue acentos', (await titulos(page)).some((t) => t.includes('autorización')), (await titulos(page)).join(' | '));
  await page.locator('#ga-buscar').fill('palabraquenoexiste'); await espera(200);
  comprobar('sin coincidencias lo dice', await page.getByText('Ningún diagrama coincide').isVisible());
  await page.locator('#ga-buscar').fill(''); await page.getByRole('tab', { name: 'Para el negocio' }).click(); await espera(300);

  // Tema oscuro: las miniaturas oscuras.
  await page.getByRole('button', { name: /Oscuro/ }).first().click(); await espera(800);
  comprobar('en oscuro se ven las miniaturas oscuras', await page.evaluate(() => {
    const o = document.querySelector('.tarjeta-diagrama .solo-oscuro');
    return getComputedStyle(o).display !== 'none' && getComputedStyle(document.querySelector('.tarjeta-diagrama .solo-claro')).display === 'none';
  }));
  await cap(page, '02-galeria-oscuro-escritorio');
  await revisar(page, 'galeria-oscuro');
  await page.getByRole('button', { name: /Claro/ }).first().click(); await espera(300);

  // Una tarjeta abre su diagrama.
  await page.locator('.tarjeta-diagrama', { hasText: 'El viaje de un ticket' }).click(); await listo(page); await espera(400);
  comprobar('una tarjeta abre el diagrama dibujado', page.url().endsWith('/diagramas/31-viaje-de-un-ticket.md')
    && await page.locator('.visor-lienzo img').evaluate((i) => i.complete && i.naturalWidth > 0));

  // «?» de Trabajos: los diagramas de esa pantalla, primero los sencillos.
  await irA(page, '/admin/trabajos');
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(500);
  const relacionados = await page.locator('dialog.panel-ayuda .relacionados button').allInnerTexts();
  comprobar('la ayuda de Trabajos ofrece sus diagramas, primero el sencillo', relacionados[0] === 'El viaje de un ticket' && relacionados.length >= 3, relacionados.join(' | '));
  await page.locator('dialog.panel-ayuda .relacionados button', { hasText: 'El viaje de un ticket' }).click(); await espera(800);
  comprobar('abrir uno lo muestra en el mismo panel', await page.locator('dialog.panel-ayuda[open]').count() === 1
    && await page.locator('dialog.panel-ayuda[open] .visor-lienzo img').evaluate((i) => i.complete && i.naturalWidth > 0).catch(() => false)
    && await page.locator('dialog.panel-ayuda[open] .relacionados').count() === 1);
  await cap(page, '03-ayuda-con-diagramas-escritorio');
  await revisar(page, 'panel-diagramas');
  await page.keyboard.press('Escape'); await espera(200);

  // Trabajo fallido: su diagrama.
  await irA(page, '/admin/trabajos?estado=fallido');
  await page.locator('tbody tr .fila-abrir').first().click(); await page.waitForSelector('dialog[open]'); await espera(300);
  const verDiagrama = page.locator('dialog[open]').getByRole('button', { name: /^Ver el diagrama:/ });
  comprobar('un trabajo fallido ofrece su diagrama', await verDiagrama.count() === 1, await verDiagrama.innerText().catch(() => ''));
  if (await verDiagrama.count()) {
    await verDiagrama.click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(500);
    comprobar('el diagrama del error se abre en el panel', await page.locator('dialog.panel-ayuda[open] .visor-lienzo img').count() === 1);
    await page.keyboard.press('Escape'); await espera(200);
  }
  await page.keyboard.press('Escape'); await espera(200);

  // Comprobador: diagrama del error y orden de autorización, sin salir a GitHub.
  await irA(page, '/admin/pruebas/permisos?aplicacion=cli_zey4vxc3gf7lhmq5');
  await page.locator('label[for="pe-llave"]').click(); await espera(900);
  const botones = await page.locator('.tarjeta-previa').getByRole('button', { name: /^Ver el diagrama:/ }).allInnerTexts();
  comprobar('el comprobador ofrece el diagrama del error y el orden de autorización', botones.length === 2 && botones.some((b) => b.includes('Identificación')), botones.join(' | '));
  comprobar('ya no enlaza a GitHub', await page.locator('.tarjeta-previa a[href*="github.com"]').count() === 0);
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ayuda/diagramas');
  await espera(800);
  await cap(page, '04-galeria-celular');
  await revisar(page, 'galeria-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 8).join('\n') : 'sin errores');
