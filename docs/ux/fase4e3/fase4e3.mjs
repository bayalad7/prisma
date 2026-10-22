// Fase 4, entrega 3: ayuda de cada pantalla, enlaces desde los errores y «Saber más». Uso: node fase4e3.mjs <carpeta-salida>
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
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
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
const panel = (page) => page.locator('dialog.panel-ayuda[open]');

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/');
  const pantallas = { '/admin/': 'Inicio', '/admin/impresoras': 'Impresoras en el panel', '/admin/solicitudes': 'Solicitudes de origen',
    '/admin/trabajos': 'Trabajos', '/admin/pruebas': 'Probar impresión', '/admin/pruebas/permisos': '¿Por qué mi app no puede imprimir?',
    '/admin/pruebas/rollo/prueba-2': 'Para quien programa una aplicación', '/admin/aplicaciones': 'Aplicaciones y solicitudes',
    '/admin/imagenes': 'Imágenes', '/admin/ajustes': 'Ajustes del panel' };
  let bien = 0;
  for (const [ruta, titulo] of Object.entries(pantallas)) {
    await irA(page, ruta);
    const boton = page.getByRole('button', { name: 'Ayuda de esta pantalla' });
    if (!(await boton.count())) { comprobar(`${ruta}: tiene botón de ayuda`, false); continue; }
    await boton.click(); await page.waitForSelector('dialog.panel-ayuda[open]', { timeout: 5000 }).catch(() => {}); await espera(300);
    const t = await panel(page).locator('h2').first().innerText().catch(() => '');
    if (t === titulo) bien++; else comprobar(`${ruta}: la ayuda abre su sección`, false, `«${t}»`);
    await page.keyboard.press('Escape'); await espera(200);
  }
  comprobar(`las ${Object.keys(pantallas).length} pantallas abren su sección del manual`, bien === Object.keys(pantallas).length, `${bien}`);
  await irA(page, '/admin/ayuda');
  comprobar('la Ayuda no tiene botón de ayuda', !(await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).count()));

  // No se pierde lo que se estaba haciendo: JSON escrito en Probar impresión.
  await irA(page, '/admin/pruebas');
  await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
  const antes = (await page.locator('#pr-json').inputValue()).replace('Gracias', 'GRACIAS-PRUEBA');
  await page.locator('#pr-json').fill(antes); await espera(300);
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(300);
  await cap(page, '01-panel-ayuda-escritorio');
  await revisar(page, 'panel-escritorio');
  await page.keyboard.press('Escape'); await espera(300);
  comprobar('cerrar la ayuda conserva lo escrito y devuelve el foco', (await page.locator('#pr-json').inputValue()) === antes
    && await page.evaluate(() => document.activeElement?.getAttribute('aria-label') === 'Ayuda de esta pantalla'));

  // «Abrir en la Ayuda» lleva a la página del manual.
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(200);
  await panel(page).getByRole('link', { name: 'Abrir en la Ayuda' }).click(); await listo(page); await espera(300);
  comprobar('«Abrir en la Ayuda» cierra el panel y abre la página', page.url().endsWith('/admin/ayuda/administracion/probar-impresion.md') && !(await panel(page).count()));

  // Detalle de un trabajo fallido: «Leer en el manual».
  await irA(page, '/admin/trabajos?estado=fallido');
  const fila = page.locator('tbody tr .fila-abrir').first();
  if (await fila.count()) {
    await fila.click(); await page.waitForSelector('dialog[open]'); await espera(300);
    await page.locator('dialog[open]').getByRole('button', { name: 'Leer en el manual' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(300);
    const t = await panel(page).locator('h2').first().innerText();
    comprobar('un trabajo fallido enlaza la sección que lo resuelve', t.length > 0, `«${t}»`);
    await cap(page, '02-trabajo-manual-escritorio');
    await page.keyboard.press('Escape'); await espera(200); await page.keyboard.press('Escape'); await espera(200);
  } else comprobar('hay un trabajo fallido para probar', false);

  // Aviso de error con «Cómo resolverlo» (se dispara desde la consola: prueba del componente).
  await page.evaluate(() => window.PrismaUI.avisoError({ message: 'El puente intentó conectarse con la impresora y no respondió.', codigo: 'IMPRESORA_INALCANZABLE' }));
  await espera(300);
  const toast = page.locator('.toast.error').last();
  comprobar('un error con código ofrece «Cómo resolverlo»', await toast.getByRole('button', { name: 'Cómo resolverlo' }).isVisible());
  await cap(page, '03-aviso-error-escritorio');
  await espera(9500);
  comprobar('ese aviso no se va solo (tiene una acción)', await toast.isVisible());
  await toast.getByRole('button', { name: 'Cómo resolverlo' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(300);
  comprobar('«Cómo resolverlo» abre «La impresora no imprime»', (await panel(page).locator('h2').first().innerText()) === 'La impresora no imprime');
  await page.keyboard.press('Escape'); await espera(200);

  // «Saber más» junto a un campo técnico, sin cerrar el formulario.
  await irA(page, '/admin/aplicaciones');
  await page.getByRole('button', { name: 'Agregar aplicación' }).first().click(); await espera(400);
  await page.locator('#cli-nombre').fill('App de la ayuda');
  await page.getByRole('button', { name: /Más información.*página/i }).first().click(); await espera(200);
  await page.getByRole('button', { name: 'Saber más' }).first().click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(300);
  comprobar('«Saber más» abre el manual sobre el formulario', await panel(page).isVisible(), await panel(page).locator('h2').first().innerText());
  await cap(page, '04-saber-mas-escritorio');
  await page.keyboard.press('Escape'); await espera(300);
  comprobar('el formulario conserva lo escrito', (await page.locator('#cli-nombre').inputValue()) === 'App de la ayuda');
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/impresoras');
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click(); await page.waitForSelector('dialog.panel-ayuda[open]'); await espera(400);
  await cap(page, '05-panel-ayuda-celular');
  await revisar(page, 'panel-celular');
  await page.keyboard.press('Escape'); await espera(200);
  await cap(page, '06-encabezado-celular');
  await revisar(page, 'impresoras-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 8).join('\n') : 'sin errores');
