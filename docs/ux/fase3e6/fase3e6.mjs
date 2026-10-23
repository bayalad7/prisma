// Fase 3, entrega 6: Ajustes y aviso de sesión. Uso: node fase3e6.mjs <carpeta-salida>
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
  await page.waitForLoadState('networkidle'); await espera(400);
}
async function abrir(browser, vp, ruta, reloj = false) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX' });
  const page = await ctx.newPage();
  if (reloj) await page.clock.install();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); }
  await listo(page);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const pestana = async (page, nombre) => { await page.getByRole('tab', { name: nombre }).click(); await espera(300); };

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ajustes');
  await cap(page, '01-general-escritorio');
  await revisar(page, 'general-escritorio');
  await page.locator('#aj-nombre').fill('');
  await page.getByLabel('Puente', { exact: true }).getByRole('button', { name: 'Guardar' }).click(); await espera(200);
  comprobar('nombre vacío: error junto al campo', await page.getByText('Escribe un nombre de 1 a 60 caracteres.').isVisible());
  await page.locator('#aj-ip').fill('-5');
  await page.getByLabel('Límite de tickets').getByRole('button', { name: 'Guardar' }).click(); await espera(200);
  comprobar('límite negativo: error junto al campo', await page.getByText('Escribe un número de 0 a 100000.').isVisible());
  await cap(page, '02-general-errores-escritorio');

  await pestana(page, 'Seguridad y acceso');
  comprobar('la pestaña queda en la dirección', page.url().endsWith('?pestana=seguridad'), page.url());
  await page.locator('#aj-actual').fill('no-es-la-contrasena');
  await page.locator('#aj-nueva').fill('una frase nueva y larga');
  await page.locator('#aj-repetir').fill('una frase nueva y larga');
  await page.getByRole('button', { name: 'Cambiar contraseña' }).click(); await espera(1500);
  comprobar('contraseña actual equivocada: error junto al campo', await page.locator('#aj-actual').getAttribute('aria-invalid') === 'true');
  await cap(page, '03-seguridad-escritorio');
  await revisar(page, 'seguridad-escritorio');
  await page.locator('label[for="aj-privada"]').click(); await page.waitForSelector('dialog'); await espera(300);
  comprobar('quitar «solo la red del negocio» pide confirmación', await page.locator('dialog').getByText('¿Permitir abrir el panel desde cualquier red?').isVisible());
  await cap(page, '04-acceso-confirmar-escritorio', false);
  await page.locator('dialog').getByRole('button', { name: 'Cancelar' }).click(); await espera(300);
  comprobar('cancelar deja el interruptor como estaba', await page.locator('#aj-privada').isChecked());

  await pestana(page, 'HTTPS');
  await cap(page, '05-https-escritorio');
  await revisar(page, 'https-escritorio');
  await page.getByRole('button', { name: 'Regenerar certificado' }).click(); await page.waitForSelector('dialog'); await espera(300);
  comprobar('regenerar dice qué pasará en cada equipo', await page.locator('dialog').getByText(/tendrá que volver a confiar/).isVisible());
  await page.locator('dialog').getByRole('button', { name: 'Cancelar' }).click(); await espera(200);

  await pestana(page, 'Respaldo');
  await page.getByRole('button', { name: 'Restaurar', exact: true }).click(); await espera(200);
  comprobar('restaurar sin archivo ni confirmación: errores junto a los campos', await page.getByText('Elige el archivo de respaldo').isVisible() && await page.getByText('Escribe RESTAURAR, en mayúsculas.').isVisible());
  await cap(page, '06-respaldo-escritorio');
  await revisar(page, 'respaldo-escritorio');
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ajustes?pestana=https');
  comprobar('?pestana=https abre esa pestaña', await page.getByRole('tab', { name: 'HTTPS' }).getAttribute('aria-selected') === 'true');
  await cap(page, '07-https-celular');
  await revisar(page, 'https-celular');
  await ctx.close();
}
{
  // Aviso de sesión con el reloj simulado del navegador.
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/', true);
  await page.clock.fastForward('28:30');
  await page.waitForSelector('dialog[role="alertdialog"]', { timeout: 5000 }).catch(() => {});
  comprobar('avisa 2 minutos antes de cerrar la sesión', await page.getByText('Tu sesión está por cerrarse').isVisible());
  await cap(page, '08-aviso-sesion-escritorio', false);
  await revisar(page, 'aviso-sesion-escritorio');
  await page.getByRole('button', { name: 'Seguir conectado' }).click(); await espera(800);
  comprobar('«Seguir conectado» cierra el aviso', !(await page.locator('dialog[role="alertdialog"]').count()));
  await page.clock.fastForward('31:00'); await espera(1500);
  comprobar('sin actividad, la sesión se cierra y pide entrar', await page.locator('#contrasena').isVisible());
  await cap(page, '09-sesion-cerrada-escritorio', false);
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
