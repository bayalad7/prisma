// Fase 6.4: comprobador de permisos. Uso: node fase64.mjs <carpeta-salida>
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
const resumen = (page) => page.locator('.tarjeta-previa .nota-caja').first().innerText().catch(() => '');

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/aplicaciones');
  await page.locator('.tarjeta').filter({ hasText: 'Mi app en desarrollo' }).first().getByRole('button', { name: 'Comprobar permisos' }).click(); await espera(1500);
  comprobar('«Comprobar permisos» abre el comprobador con la aplicación', page.url().includes('/admin/pruebas/permisos?aplicacion=')
    && (await page.locator('#pe-app option:checked').innerText()) === 'Mi app en desarrollo');
  comprobar('su caso típico puede imprimir', /^Puede imprimir/.test(await resumen(page)), await resumen(page));
  await cap(page, '01-todo-bien-escritorio');
  await revisar(page, 'todo-bien-escritorio');

  await page.locator('label[for="pe-llave"]').click(); await espera(800);
  comprobar('sin llave: 401 LLAVE_REQUERIDA y los pasos siguientes no se revisan', (await resumen(page)).includes('401 LLAVE_REQUERIDA')
    && await page.locator('.paso-pendiente').count() === 4);
  await cap(page, '02-sin-llave-escritorio');
  await page.locator('label[for="pe-llave"]').click(); await espera(500);

  await page.getByText('Otra página web').click(); await espera(200);
  await page.locator('#pe-otra').fill('https://desconocido-6-4.example'); await espera(1200);
  comprobar('origen desconocido: 403 ORIGEN_NO_AUTORIZADO, sin decir que quedó pendiente', (await resumen(page)).includes('403 ORIGEN_NO_AUTORIZADO')
    && !(await resumen(page)).includes('quedó como solicitud pendiente') && await page.getByRole('button', { name: 'Editar la aplicación' }).isVisible());
  const solicitudes = await page.evaluate(() => fetch('/admin/api/solicitudes').then((r) => r.text()));
  comprobar('el comprobador no registra la solicitud', !solicitudes.includes('desconocido-6-4'));
  await cap(page, '03-origen-desconocido-escritorio');
  await revisar(page, 'origen-escritorio');

  await page.locator('#pe-app').selectOption({ label: 'Menú digital' }); await espera(1000);
  comprobar('al cambiar de aplicación propone su página y una impresora permitida', (await page.locator('#pe-impresora').inputValue()) === 'cocina'
    && /^Puede imprimir/.test(await resumen(page)), await resumen(page));
  await page.locator('#pe-impresora').selectOption('caja'); await espera(800);
  comprobar('impresora no permitida: 403 IMPRESORA_NO_PERMITIDA', (await resumen(page)).includes('IMPRESORA_NO_PERMITIDA')
    && await page.getByRole('button', { name: 'Editar la aplicación' }).isVisible());
  await cap(page, '04-impresora-escritorio');

  await page.locator('#pe-app').selectOption({ label: 'Punto de venta web' }); await espera(900);
  await page.locator('label[for="pe-cajon"]').click(); await espera(700);
  comprobar('abrir_cajon permitido a «Punto de venta web»', /^Puede imprimir/.test(await resumen(page)), await resumen(page));
  await page.locator('label[for="pe-raw"]').click(); await espera(700);
  comprobar('raw sin permiso: OPERACION_NO_PERMITIDA', (await resumen(page)).includes('OPERACION_NO_PERMITIDA'));

  await page.locator('#pe-app').selectOption({ label: 'Tienda 2540' }); await espera(900);
  await page.getByText('Otra página web').click(); await page.locator('#pe-otra').fill('https://dev.ejemplo.mx'); await espera(1200);
  comprobar('origen de otra aplicación con su llave: LLAVE_INVALIDA', (await resumen(page)).includes('LLAVE_INVALIDA'), await resumen(page));
  await page.locator('label[for="pe-llave"]').click(); await espera(800);
  comprobar('sin llave en el origen de otra: imprimiría como esa otra', (await resumen(page)).includes('LLAVE_REQUERIDA') || (await resumen(page)).includes('a nombre de'),
    await resumen(page));
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/pruebas');
  await page.getByRole('button', { name: '¿Por qué mi app no puede imprimir?' }).click(); await espera(1200);
  comprobar('«Probar impresión» lleva al comprobador', page.url().endsWith('/admin/pruebas/permisos'));
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/pruebas/permisos');
  await cap(page, '05-comprobador-celular');
  await revisar(page, 'comprobador-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
