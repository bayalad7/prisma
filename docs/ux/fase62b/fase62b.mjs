// Fase 6.2b: constructor visual de «Probar impresión». Uso: node fase62b.mjs <carpeta-salida>
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
async function abrir(browser, vp, ruta) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(800);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const json = async (page) => JSON.parse(await page.locator('#pr-json').inputValue());
const valido = (page) => page.waitForSelector('.estado-valido', { timeout: 8000 }).then(() => true).catch(() => false);

const browser = await chromium.launch();
for (const vp of ['escritorio', 'celular']) {
  const { ctx, page } = await abrir(browser, vp, '/admin/pruebas');
  await page.getByRole('radio', { name: /^Comanda/ }).check({ force: true }); await valido(page);
  await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
  await page.getByRole('tab', { name: 'Constructor' }).click(); await espera(600);
  const n = await page.locator('details.operacion').count();
  comprobar(`${vp}: el constructor muestra las operaciones de la plantilla`, n === (await json(page)).operaciones.length, `${n}`);
  await cap(page, `01-constructor-${vp}`);
  await revisar(page, `constructor-${vp}`);

  if (vp === 'escritorio') {
    // Editar el texto de la operación 2.
    await page.locator('#pr-op-1-resumen').click(); await espera(200);
    await page.locator('#pr-op-1-valor').fill('MESA 7');
    await espera(900);
    comprobar('editar un campo cambia el JSON y la vista previa', (await json(page)).operaciones[1].valor === 'MESA 7' && await valido(page));
    await page.locator('#pr-op-1-tamano').selectOption({ label: 'Normal' }); await espera(700);
    comprobar('tamaño «Normal» quita el campo (valor por defecto)', (await json(page)).operaciones[1].tamano === undefined);
    await cap(page, '02-editar-escritorio');
    await revisar(page, 'editar-escritorio');

    // Agregar un QR, subirlo y eliminar el separador.
    await page.locator('#pr-agregar-tipo').selectOption('qr');
    await page.getByRole('button', { name: 'Agregar', exact: true }).click(); await espera(400);
    const ops = (await json(page)).operaciones;
    comprobar('agregar pone la operación al final, abierta y con foco', ops[ops.length - 1].tipo === 'qr'
      && await page.evaluate(() => document.activeElement.id) === `pr-op-${ops.length - 1}-resumen`);
    await page.locator(`#pr-op-${ops.length - 1}-valor`).fill('https://minegocio.mx/p/7');
    await page.locator(`#pr-op-${ops.length - 1}-subir`).click(); await espera(400);
    const ops2 = (await json(page)).operaciones;
    comprobar('subir mueve la operación y conserva el foco', ops2[ops2.length - 2].tipo === 'qr' && ops2[ops2.length - 2].valor === 'https://minegocio.mx/p/7'
      && await page.evaluate(() => document.activeElement.id) === `pr-op-${ops.length - 2}-subir`);
    const antes = ops2.length;
    await page.locator('#pr-op-3-resumen').click(); await espera(200);
    const tipo3 = ops2[3].tipo;
    await page.getByRole('button', { name: 'Eliminar la operación 4' }).click(); await espera(400);
    comprobar('eliminar quita la operación', (await json(page)).operaciones.length === antes - 1, tipo3);
    comprobar('la vista previa sigue válida', await valido(page));
    await cap(page, '03-agregar-mover-escritorio');

    // Una fila: agregar una columna.
    await page.locator('#pr-agregar-tipo').selectOption('fila');
    await page.getByRole('button', { name: 'Agregar', exact: true }).click(); await espera(400);
    const k = (await json(page)).operaciones.length - 1;
    await page.locator(`#pr-op-${k}-agregar-columna`).click(); await espera(400);
    await page.locator(`#pr-op-${k}-c2-texto`).fill('$99.00');
    await page.locator(`#pr-op-${k}-c0-ancho`).fill('20'); await espera(900);
    const fila = (await json(page)).operaciones[k];
    comprobar('fila: columnas agregadas y con ancho', fila.columnas.length === 3 && fila.columnas[2].texto === '$99.00' && fila.columnas[0].ancho === 20);
    comprobar('fila: la vista previa sigue válida', await valido(page));
    await cap(page, '04-fila-escritorio');
    await revisar(page, 'fila-escritorio');

    // Ida y vuelta con JSON: lo hecho en el constructor está en JSON y al revés.
    await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
    const texto = await page.locator('#pr-json').inputValue();
    await page.locator('#pr-json').fill(texto.replace('"MESA 7"', '"MESA 9"')); await espera(900);
    await page.getByRole('tab', { name: 'Constructor' }).click(); await espera(500);
    await page.locator('#pr-op-1-resumen').click(); await espera(200);
    comprobar('cambiar de modo conserva el documento en ambos sentidos', await page.locator('#pr-op-1-valor').inputValue() === 'MESA 9');
    await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
    await page.locator('#pr-json').fill('{ "version": '); await espera(700);
    await page.getByRole('tab', { name: 'Constructor' }).click(); await espera(400);
    comprobar('con el JSON roto, el constructor lo explica', await page.getByText('El JSON tiene un error').isVisible());
    await cap(page, '05-json-roto-escritorio', false);
    await page.getByRole('button', { name: 'Ir a JSON' }).click(); await espera(200);
    comprobar('«Ir a JSON» abre la pestaña', await page.getByRole('tab', { name: 'JSON' }).getAttribute('aria-selected') === 'true');
  } else {
    await page.locator('#pr-op-0-resumen').click(); await espera(300);
    await cap(page, '02-editar-celular');
    await revisar(page, 'editar-celular');
  }
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
