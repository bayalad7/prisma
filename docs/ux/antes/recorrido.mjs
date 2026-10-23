// Recorre el panel real de PRISMA 0.1.0 y captura cada pantalla y estado en escritorio y celular.
// Uso: node recorrido.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const DEMO = 'https://127.0.0.1:18443/admin/';
const VACIO = 'https://127.0.0.1:8444/admin/';
const CLAVE = 'contrasena-demo-2026';
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};
fs.mkdirSync(SALIDA, { recursive: true });
const axeTodo = {};
const bitacora = [];
const espera = (ms) => new Promise((r) => setTimeout(r, ms));

async function cap(page, vp, nombre, { completa = true, el } = {}) {
  const archivo = path.join(SALIDA, `${nombre}-${vp}.png`);
  if (el) await el.screenshot({ path: archivo });
  else await page.screenshot({ path: archivo, fullPage: completa });
  bitacora.push(`${nombre}-${vp}.png`);
}
async function listo(page) {
  await page.waitForSelector('#cargando', { state: 'detached', timeout: 20000 }).catch(() => {});
  await page.waitForLoadState('networkidle').catch(() => {});
  await espera(250);
}
async function ir(page, base, seccion) {
  await page.goto(`${base}#${seccion}`);
  await page.evaluate(() => window.dispatchEvent(new HashChangeEvent('hashchange')));
  await listo(page);
}
async function axe(page, vp, nombre) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  axeTodo[`${nombre}-${vp}`] = r.violations.map((v) => ({
    id: v.id, impacto: v.impact, ayuda: v.help, nodos: v.nodes.length,
    ejemplos: v.nodes.slice(0, 3).map((n) => n.target.join(' ')),
  }));
}
async function cerrarDialogo(page) {
  await page.keyboard.press('Escape');
  await page.waitForSelector('dialog', { state: 'detached' }).catch(() => {});
}
async function pngDePrueba(browser) {
  const p = await browser.newPage({ viewport: { width: 300, height: 120 } });
  await p.setContent('<div id="l" style="width:300px;height:120px;display:grid;place-items:center;background:#fff;font:bold 44px sans-serif">LA ESPERANZA</div>');
  const ruta = path.join(SALIDA, '..', 'logo-prueba.png');
  await p.locator('#l').screenshot({ path: ruta });
  await p.close();
  return ruta;
}

async function recorrerVacio(browser, vp) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  await page.goto(VACIO); await listo(page);
  if (await page.locator('#nueva').count()) {
    await cap(page, vp, '01-configuracion-inicial');
    await axe(page, vp, '01-configuracion-inicial');
    await page.fill('#nueva', 'corta'); await page.fill('#repetir', 'corta');
    await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
    await espera(200);
    await cap(page, vp, '02-configuracion-inicial-error');
    await page.fill('#nueva', CLAVE); await page.fill('#repetir', CLAVE);
    await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
  } else {
    await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click();
  }
  await listo(page);
  await cap(page, vp, '03-inicio-vacio');
  for (const [s, n] of [['impresoras', '04-impresoras-vacio'], ['clientes', '05-aplicaciones-vacio'], ['solicitudes', '06-solicitudes-vacio'], ['trabajos', '07-trabajos-vacio'], ['recursos', '08-imagenes-vacio']]) {
    await ir(page, VACIO, s); await cap(page, vp, n);
  }
  await ctx.close();
}

async function recorrerDemo(browser, vp, logo) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  await page.goto(DEMO); await listo(page);
  if (await page.locator('#nueva').count()) {
    await page.fill('#nueva', CLAVE); await page.fill('#repetir', CLAVE);
    await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
    await listo(page);
    // Cerrar sesión para capturar el inicio de sesión.
    if (vp === 'celular') await page.getByRole('button', { name: 'Menú' }).click();
    await page.getByRole('button', { name: 'Cerrar sesión' }).click();
    await listo(page);
  }
  await cap(page, vp, '10-entrar');
  await axe(page, vp, '10-entrar');
  await page.fill('#contrasena', 'equivocada'); await page.getByRole('button', { name: 'Entrar' }).click();
  await espera(1500);
  await cap(page, vp, '11-entrar-error');
  await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click();
  await listo(page);

  // Inicio
  await cap(page, vp, '12-inicio');
  await axe(page, vp, '12-inicio');
  if (vp === 'celular') {
    await page.getByRole('button', { name: 'Menú' }).click(); await espera(200);
    await cap(page, vp, '13-menu-abierto', { completa: false });
    await page.getByRole('button', { name: 'Menú' }).click();
  } else {
    // Foco visible: tabula hasta el tercer enlace del menú.
    await page.keyboard.press('Tab'); await page.keyboard.press('Tab'); await page.keyboard.press('Tab');
    await cap(page, vp, '13-foco-teclado', { completa: false });
  }

  // Impresoras
  await ir(page, DEMO, 'impresoras');
  await cap(page, vp, '20-impresoras');
  await axe(page, vp, '20-impresoras');
  const tarjeta = (alias) => page.locator('article').filter({ has: page.locator('.alias', { hasText: new RegExp(`^${alias}$`) }) });
  await tarjeta('barra').getByRole('button', { name: 'Probar conexión' }).click();
  await tarjeta('barra').locator('.resultado:not([hidden])').waitFor({ timeout: 15000 }).catch(() => {});
  await cap(page, vp, '21-impresora-probar-error', { el: tarjeta('barra') });
  await tarjeta('caja').getByRole('button', { name: 'Ticket de diagnóstico' }).click();
  await page.waitForSelector('dialog', { timeout: 20000 }); await espera(300);
  await cap(page, vp, '22-diagnostico', { completa: false });
  await axe(page, vp, '22-diagnostico');
  await cerrarDialogo(page);
  await tarjeta('caja').getByRole('button', { name: 'Probar páginas de códigos' }).click();
  await page.waitForSelector('dialog', { timeout: 20000 }); await espera(300);
  await cap(page, vp, '23-paginas-codigos', { completa: false });
  await cerrarDialogo(page);
  await tarjeta('barra').getByRole('button', { name: 'Ticket de diagnóstico' }).click();
  await tarjeta('barra').locator('.resultado:not([hidden])').waitFor({ timeout: 20000 }).catch(() => {});
  await cap(page, vp, '24-diagnostico-error', { el: tarjeta('barra') });

  // Asistente de alta
  await ir(page, DEMO, 'impresoras');
  await page.getByRole('button', { name: 'Agregar impresora' }).first().click();
  await page.waitForFunction(() => !document.body.textContent.includes('Buscando…'), null, { timeout: 20000 }).catch(() => {});
  await espera(300);
  await cap(page, vp, '25-asistente-paso1');
  await axe(page, vp, '25-asistente-paso1');
  await page.fill('#destino-manual', '127.0.0.1:9100');
  await page.getByRole('button', { name: 'Continuar' }).dispatchEvent('click'); await espera(300);
  await cap(page, vp, '26-asistente-paso2');
  await axe(page, vp, '26-asistente-paso2');
  await page.getByRole('button', { name: 'Guardar impresora' }).click(); await espera(400);
  await cap(page, vp, '27-asistente-alias-vacio', { completa: false });
  await page.getByRole('button', { name: 'Probar conexión' }).click();
  await page.locator('.resultado:not([hidden])').waitFor({ timeout: 15000 }).catch(() => {});
  await cap(page, vp, '28-asistente-probado');
  await page.locator('details.avanzado summary').click();
  await cap(page, vp, '29-asistente-avanzado');
  await ir(page, DEMO, 'impresoras');
  await tarjeta('cocina').getByRole('button', { name: 'Editar' }).click(); await espera(300);
  await cap(page, vp, '30-impresora-editar');

  // Aplicaciones
  await ir(page, DEMO, 'clientes');
  await cap(page, vp, '40-aplicaciones');
  await axe(page, vp, '40-aplicaciones');
  await page.locator('article').first().getByRole('button', { name: 'Código de integración' }).click();
  await page.waitForSelector('dialog'); await espera(200);
  await cap(page, vp, '41-codigo-integracion', { completa: false });
  await cerrarDialogo(page);
  await page.getByRole('button', { name: 'Agregar aplicación' }).click(); await espera(300);
  await cap(page, vp, '42-aplicacion-nueva');
  await axe(page, vp, '42-aplicacion-nueva');
  await page.locator('input[name=nivel][value="3"]').check({ force: true }); await espera(200);
  await cap(page, vp, '43-aplicacion-nivel3');
  await page.locator('input[name=nivel][value="2"]').check({ force: true });
  await page.fill('#cli-nombre', `Tienda de prueba ${vp} ${Date.now() % 1000}`);
  await page.fill('#cli-origen', `https://tienda-${vp}-${Date.now()}.ejemplo.mx`);
  await page.getByRole('button', { name: 'Crear aplicación' }).click();
  await page.waitForSelector('dialog', { timeout: 10000 }); await espera(300);
  await cap(page, vp, '44-llave-nueva', { completa: false });
  await axe(page, vp, '44-llave-nueva');
  await page.getByRole('button', { name: 'Ya la copié' }).click();

  // Solicitudes
  await ir(page, DEMO, 'solicitudes');
  await cap(page, vp, '50-solicitudes');
  await axe(page, vp, '50-solicitudes');
  await page.getByRole('button', { name: 'Aprobar' }).first().click();
  await page.waitForSelector('dialog'); await espera(200);
  await cap(page, vp, '51-aprobar-origen', { completa: false });
  await cerrarDialogo(page);

  // Trabajos
  await ir(page, DEMO, 'trabajos');
  await cap(page, vp, '60-trabajos');
  await axe(page, vp, '60-trabajos');
  await page.selectOption('#f-estado', 'fallido'); await espera(600);
  await cap(page, vp, '61-trabajos-fallidos');

  // Imágenes
  await ir(page, DEMO, 'recursos');
  await cap(page, vp, '70-imagenes');
  await axe(page, vp, '70-imagenes');
  await page.setInputFiles('#rec-archivo', logo); await espera(800);
  await cap(page, vp, '71-imagen-vista-previa');

  // Ajustes y Ayuda
  await ir(page, DEMO, 'ajustes');
  await cap(page, vp, '80-ajustes');
  await axe(page, vp, '80-ajustes');
  await ir(page, DEMO, 'ayuda');
  await cap(page, vp, '90-ayuda');
  await axe(page, vp, '90-ayuda');

  // Estados: cargando y error
  await page.route('**/admin/api/resumen', async (r) => { await espera(4000); await r.continue(); });
  await page.goto(`${DEMO}#inicio`); await page.evaluate(() => window.dispatchEvent(new HashChangeEvent('hashchange')));
  await espera(600);
  await cap(page, vp, '95-cargando', { completa: false });
  await page.unroute('**/admin/api/resumen');
  await page.route('**/admin/api/trabajos*', (r) => r.fulfill({ status: 500, contentType: 'application/json',
    body: JSON.stringify({ ok: false, codigo: 'ERROR_INTERNO', mensaje: 'Ocurrió un error interno en el puente.', sugerencia: 'Inténtalo de nuevo; si persiste, revisa la bitácora.' }) }));
  await ir(page, DEMO, 'trabajos');
  await cap(page, vp, '96-error-api', { completa: false });
  await page.unroute('**/admin/api/trabajos*');
  await page.route('**/admin/api/sesion', (r) => r.abort());
  await page.goto(DEMO); await espera(800);
  await cap(page, vp, '97-sin-conexion', { completa: false });
  await page.unroute('**/admin/api/sesion');
  await ctx.close();
}

const browser = await chromium.launch();
const logo = await pngDePrueba(browser);
const SOLO = process.argv[3]; if (!SOLO) for (const vp of Object.keys(VISTAS)) await recorrerVacio(browser, vp);
for (const vp of Object.keys(VISTAS)) if (!SOLO || SOLO === vp) await recorrerDemo(browser, vp, logo);
await browser.close();
fs.writeFileSync(path.join(SALIDA, '..', `axe-antes${SOLO ? '-' + SOLO : ''}.json`), JSON.stringify(axeTodo, null, 2));
console.log(bitacora.length, 'capturas');
