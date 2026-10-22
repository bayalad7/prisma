// Fase 3, entrega 4: Aplicaciones y Solicitudes. Uso: node fase3e4.mjs <carpeta-salida>
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
const n = Date.now() % 10000;

async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((x) => x.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', acceptDownloads: true });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await listo(page);
  return { ctx, page };
}
async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(400);
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const cerrar = async (page) => { await page.keyboard.press('Escape'); await page.waitForSelector('dialog', { state: 'detached' }).catch(() => {}); };

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/aplicaciones');
  await cap(page, '01-aplicaciones-escritorio');
  await revisar(page, 'aplicaciones-escritorio');

  // Alta con errores.
  await page.getByRole('button', { name: 'Agregar aplicación' }).first().click(); await espera(400);
  await cap(page, '02-nueva-escritorio');
  await revisar(page, 'nueva-escritorio');
  await page.locator('label.opcion', { hasText: 'Solo la dirección de la página' }).click();
  await page.getByRole('button', { name: 'Crear aplicación' }).click(); await espera(300);
  comprobar('sin nombre: error junto al campo', await page.getByText('Escribe el nombre de la aplicación.').isVisible());
  comprobar('nivel 1 sin página: error junto al campo', await page.getByText('hace falta al menos una página web').isVisible());
  await page.locator('#cli-origen').fill('pos.mitienda.mx/caja');
  await page.getByRole('button', { name: 'Agregar', exact: true }).click(); await espera(200);
  comprobar('origen mal escrito: explica el formato', await page.getByText(/Escribe la dirección completa, con https/).isVisible());
  await cap(page, '03-nueva-errores-escritorio');
  await page.locator('label.opcion', { hasText: 'Dirección y llave secreta' }).click();
  await page.locator('#cli-nombre').fill(`Tienda ${n}`);
  await page.locator('#cli-origen').fill(`https://tienda-${n}.ejemplo.mx`);
  await page.getByRole('button', { name: 'Crear aplicación' }).click();
  await page.waitForSelector('dialog'); await espera(500);
  await cap(page, '04-llave-escritorio', false);
  await revisar(page, 'llave-escritorio');
  const [descarga] = await Promise.all([page.waitForEvent('download'), page.getByRole('link', { name: 'Descargar como archivo' }).click()]);
  const contenido = fs.readFileSync(await descarga.path(), 'utf8');
  comprobar('descarga la llave en un archivo de texto', descarga.suggestedFilename().startsWith('llave-tienda-') && /Llave: pk_/.test(contenido), descarga.suggestedFilename());
  comprobar('el código usa el SDK real', (await page.locator('dialog pre').first().textContent()).includes('cdn.jsdelivr.net/gh/bayalad7/prisma'));
  await page.getByRole('button', { name: 'Ya la guardé' }).click(); await listo(page);

  // Código para integrar con selector de impresora.
  const tarjeta = page.locator('section.tarjeta').filter({ hasText: `Tienda ${n}` });
  await tarjeta.getByRole('button', { name: 'Código para integrar' }).click(); await page.waitForSelector('dialog'); await espera(300);
  const antes = await page.locator('dialog pre').first().textContent();
  const opciones = await page.locator('dialog select option').allTextContents();
  const actual = await page.locator('dialog select').inputValue().catch(() => '');
  const valores = await page.locator('dialog select option').evaluateAll((os) => os.map((o) => o.value));
  const otra = valores.find((v) => v !== actual);
  if (otra) await page.locator('dialog select').selectOption(otra);
  await espera(200);
  comprobar('cambiar la impresora cambia el código', opciones.length < 2 || (await page.locator('dialog pre').first().textContent()).includes(`"${otra}"`), otra);
  await cap(page, '05-codigo-escritorio', false);
  await revisar(page, 'codigo-escritorio');
  await cerrar(page);

  // Confirmar desactivar con sus consecuencias.
  await tarjeta.getByRole('button', { name: 'Desactivar' }).click(); await page.waitForSelector('dialog'); await espera(300);
  comprobar('desactivar pide confirmación y dice qué pasa', await page.locator('dialog').getByText('Los tickets que envíe se rechazarán.').isVisible());
  await cap(page, '06-desactivar-escritorio', false);
  await page.locator('dialog').getByRole('button', { name: 'Cancelar' }).click(); await espera(200);

  // Solicitudes.
  await page.goto(`${B}/admin/solicitudes`); await listo(page);
  await cap(page, '07-solicitudes-escritorio');
  await revisar(page, 'solicitudes-escritorio');
  comprobar('marca el sitio de Internet', await page.getByText('Sitio de Internet').first().isVisible());
  comprobar('marca el equipo de la red sin cifrar', await page.getByText('Equipo de tu red').first().isVisible() && await page.getByText('Sin cifrar (http)').first().isVisible());
  const internet = page.locator('article').filter({ hasText: 'Sitio de Internet' }).first();
  await internet.getByRole('button', { name: 'Aprobar' }).click(); await page.waitForSelector('dialog'); await espera(300);
  comprobar('aprobar un sitio de Internet advierte el riesgo', await page.locator('dialog').getByText(/apruébalo solo si es tuyo/).isVisible());
  await cap(page, '08-aprobar-escritorio', false);
  await revisar(page, 'aprobar-escritorio');
  await cerrar(page);

  // Eliminar impresora: dice qué aplicaciones la usan.
  await page.goto(`${B}/admin/impresoras`); await listo(page);
  await page.locator('article').filter({ has: page.locator('.alias', { hasText: /^caja$/ }) }).getByRole('button', { name: 'Eliminar' }).click();
  await page.waitForSelector('dialog'); await espera(300);
  comprobar('eliminar una impresora lista las aplicaciones afectadas', await page.locator('dialog').getByText(/Estas aplicaciones ya no podrán imprimir/).isVisible());
  await cap(page, '09-eliminar-impresora-escritorio', false);
  await page.locator('dialog').getByRole('button', { name: 'Cancelar' }).click();
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/aplicaciones');
  await cap(page, '10-aplicaciones-celular');
  await revisar(page, 'aplicaciones-celular');
  await page.goto(`${B}/admin/solicitudes`); await listo(page);
  await cap(page, '11-solicitudes-celular');
  await revisar(page, 'solicitudes-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
