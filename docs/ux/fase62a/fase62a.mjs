// Fase 6.2a: Probar impresión (plantillas, JSON, vista previa, enviar, copiar como código). Uso: node fase62a.mjs <carpeta-salida>
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
async function abrir(browser, vp, ruta, opciones = {}) {
  const ctx = await browser.newContext({ ...VISTAS[vp], ignoreHTTPSErrors: true, locale: 'es-MX', ...opciones });
  const page = await ctx.newPage();
  page.on('console', (m) => { if (['error', 'warning'].includes(m.type()) && !/Failed to load resource/.test(m.text())) informe.consola.push(`${vp}: ${m.text()}`); });
  page.on('pageerror', (e) => informe.consola.push(`${vp} pageerror: ${e.message}`));
  await page.goto(`${B}${ruta}`); await espera(800);
  if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar', exact: true }).click(); }
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15000 }).catch(() => {});
  await page.waitForLoadState('networkidle'); await espera(600);
  return { ctx, page };
}
const cap = (page, nombre, completa = true) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const previaLista = (page) => page.waitForSelector('.previa-prueba img', { timeout: 8000 }).catch(() => {});

const browser = await chromium.launch();
for (const vp of ['escritorio', 'celular']) {
  const { ctx, page } = await abrir(browser, vp, '/admin/pruebas', vp === 'escritorio' ? { permissions: ['clipboard-read', 'clipboard-write'] } : {});
  await previaLista(page);
  comprobar(`${vp}: con una virtual empieza con la venta y su vista previa`, await page.locator('.previa-prueba img').count() === 1
    && await page.getByRole('radio', { name: /^Venta Un ticket/ }).isChecked());
  comprobar(`${vp}: «Enviar a» nombra la impresora virtual`, await page.getByRole('button', { name: /^Enviar a «prueba-2»/ }).isVisible());
  await cap(page, `01-plantilla-${vp}`);
  await revisar(page, `plantilla-${vp}`);

  if (vp === 'escritorio') {
    await page.getByRole('radio', { name: /^Todas las operaciones/ }).check({ force: true }); await espera(1200);
    comprobar('plantilla con imagen faltante: aviso y error explicado', await page.getByText(/Usa la imagen guardada «logo-tienda»/).isVisible()
      && await page.getByText('No se puede dibujar el ticket').isVisible());
    await cap(page, '02-imagen-faltante-escritorio');

    await page.getByRole('radio', { name: /^Comanda/ }).check({ force: true }); await previaLista(page); await espera(500);
    await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
    const texto = await page.locator('#pr-json').inputValue();
    comprobar('cambiar a JSON conserva el documento', texto.includes('"COMANDA"') && texto.includes('"impresora": "prueba-2"'));
    await page.locator('#pr-json').fill(texto.replace('"tipo": "texto"', '"tipo": "textoo"')); await espera(1500);
    comprobar('un tipo inválido se señala con la operación', await page.getByText(/Operación 1/).first().isVisible(), await page.locator('#pr-estado').innerText());
    await cap(page, '03-json-error-escritorio');
    await revisar(page, 'json-error-escritorio');
    await page.locator('#pr-json').fill(texto.replace('"version": "1",', '"version": "1"')); await espera(800);
    comprobar('error de sintaxis con renglón y columna', await page.getByText(/no es válido en el renglón \d+, columna \d+/).isVisible(), await page.locator('#pr-estado').innerText());
    await page.locator('#pr-json').fill(texto); await previaLista(page); await espera(800);
    await page.getByRole('tab', { name: 'Plantillas' }).click(); await espera(200);
    await page.getByRole('tab', { name: 'JSON' }).click(); await espera(200);
    comprobar('volver a JSON conserva lo escrito', (await page.locator('#pr-json').inputValue()) === texto);

    await page.getByRole('button', { name: /^Enviar a «prueba-2»/ }).click(); await page.waitForSelector('.pila .nota', { timeout: 10000 }).catch(() => {}); await espera(400);
    comprobar('enviar a la virtual: resultado y respuesta de la API', await page.getByText('«prueba-2» recibió el ticket.').isVisible()
      && await page.getByText('Respuesta de la API, como la recibiría tu aplicación').isVisible());
    await page.getByText('Respuesta de la API, como la recibiría tu aplicación').click(); await espera(200);
    comprobar('la respuesta trae trabajo_id y advertencias', /"trabajo_id": "trb_/.test(await page.locator('details pre').last().innerText()));
    await cap(page, '04-enviado-escritorio');
    await revisar(page, 'enviado-escritorio');
    await page.getByRole('button', { name: 'Ver el ticket que recibió' }).click(); await page.waitForSelector('dialog img'); await espera(300);
    await cap(page, '05-ticket-recibido-escritorio', false);
    await page.locator('dialog').getByRole('button', { name: /Cerrar/ }).first().click(); await espera(300);

    await page.getByRole('button', { name: 'Copiar como código' }).click(); await page.waitForSelector('dialog pre'); await espera(300);
    const sdk = await page.locator('dialog pre').first().innerText();
    comprobar('copiar como código: SDK con la impresora y las operaciones', sdk.includes('prisma.imprimir("prueba-2", [') && sdk.includes('"COMANDA"') && sdk.includes('referencia:'));
    await cap(page, '06-codigo-escritorio', false);
    await revisar(page, 'codigo-escritorio');
    await page.locator('dialog').getByRole('tab', { name: 'curl' }).click(); await espera(200);
    comprobar('copiar como código: curl con id nuevo', /\/v1\/imprimir/.test(await page.locator('dialog pre').last().innerText()));
    await page.keyboard.press('Escape'); await espera(300);

    await page.locator('#pr-impresora').selectOption('barra'); await espera(1500);
    comprobar('cambiar de impresora cambia «Enviar a» y el documento', await page.getByRole('button', { name: /^Enviar a «barra/ }).isVisible()
      && (await page.locator('#pr-json').inputValue()).includes('"impresora": "barra"'));
    await page.getByRole('button', { name: /^Enviar a «barra/ }).click(); await page.waitForSelector('.nota.nota-error, .nota[class*=error]', { timeout: 70000 }).catch(() => {}); await espera(400);
    comprobar('impresora física que no responde: error explicado', await page.getByText('No se imprimió en «barra»').isVisible());
    await cap(page, '07-fallo-escritorio');
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
