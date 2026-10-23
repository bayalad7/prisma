// Fase 4, entrega 4: «Mi impresora no imprime», desde el diagrama 23. Uso: node fase4e4.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const espera = (ms) => new Promise((r) => setTimeout(r, ms));
const informe = { axe: {}, desborde: {}, consola: [], comprobaciones: [], caminos: [] };
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
const cap = (page, nombre, completa = false) => page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
const respuestas = (page) => page.locator('.pregunta-arbol .acciones button').allInnerTexts();
const titulo = (page) => page.locator('.vista h2').last().innerText();

const browser = await chromium.launch();
{
  const { ctx, page } = await abrir(browser, 'escritorio', '/admin/ayuda');
  await page.getByRole('link', { name: 'Empezar' }).click(); await listo(page); await espera(300);
  comprobar('la Ayuda lleva a «Mi impresora no imprime»', page.url().endsWith('/admin/ayuda/no-imprime')
    && (await titulo(page)) === '¿Sale el Ticket de prueba del panel (Impresoras)?');
  await cap(page, '01-primera-pregunta-escritorio');
  await revisar(page, 'pregunta-escritorio');

  // Recorre todos los caminos: en cada pregunta, cada respuesta; vuelve con «Cambiar».
  const finales = [];
  async function explorar(nivel, camino) {
    const opciones = await respuestas(page);
    if (opciones.length === 0) {
      const solucion = page.getByRole('button', { name: /^Cómo resolverlo:/ });
      const tiene = await solucion.count() === 1;
      let seccion = '';
      if (tiene) {
        await solucion.click(); await page.waitForSelector('dialog.panel-ayuda[open]', { timeout: 5000 }).catch(() => {}); await espera(250);
        seccion = await page.locator('dialog.panel-ayuda[open] h2').first().innerText().catch(() => '');
        await page.keyboard.press('Escape'); await espera(150);
      }
      finales.push({ camino: camino.join(' → '), solucion: tiene, seccion, cuerpo: await page.locator('dialog.panel-ayuda').count() });
      return;
    }
    for (const r of opciones) {
      await page.locator('.pregunta-arbol .acciones button', { hasText: r }).first().click(); await espera(150);
      await explorar(nivel + 1, [...camino, r]);
      // Volver a esta pregunta: «Cambiar» en su lugar del recorrido.
      await page.locator('.recorrido-arbol li').nth(nivel).getByRole('button', { name: 'Cambiar' }).click(); await espera(150);
    }
  }
  await explorar(0, []);
  informe.caminos = finales;
  comprobar(`los ${finales.length} caminos terminan en una solución con su sección del manual`,
    finales.length >= 10 && finales.every((x) => x.solucion && x.seccion), finales.filter((x) => !x.seccion).map((x) => x.camino).join(' | '));

  // Un camino completo, con la sugerencia de la impresora virtual.
  await page.getByRole('button', { name: 'Sí', exact: true }).click(); await espera(200);
  comprobar('sugiere probar con la impresora virtual', (await titulo(page)).includes('impresora virtual')
    && await page.getByRole('button', { name: 'Probar sin impresora' }).isVisible());
  await cap(page, '02-virtual-escritorio');
  await page.getByRole('button', { name: 'No', exact: true }).click(); await espera(200);
  await page.getByRole('button', { name: 'No: error de red' }).click(); await espera(200);
  await page.getByRole('button', { name: 'Aviso de certificado' }).click(); await espera(300);
  comprobar('llega a la solución con el recorrido a la vista', await page.getByText('Lo más probable').isVisible()
    && await page.locator('.recorrido-arbol li').count() === 4 && await page.evaluate(() => document.activeElement?.tagName === 'H2'));
  await cap(page, '03-solucion-escritorio');
  await revisar(page, 'solucion-escritorio');
  await page.getByRole('button', { name: 'Empezar de nuevo' }).click(); await espera(200);
  comprobar('«Empezar de nuevo» vuelve a la primera pregunta', (await titulo(page)).startsWith('¿Sale el Ticket de prueba'));
  await ctx.close();
}
{
  const { ctx, page } = await abrir(browser, 'celular', '/admin/ayuda/no-imprime');
  await page.getByRole('button', { name: 'No', exact: true }).click(); await espera(200);
  await cap(page, '04-pregunta-celular');
  await revisar(page, 'pregunta-celular');
  await ctx.close();
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
console.log(informe.comprobaciones.join('\n'));
console.log(informe.caminos.map((x) => `${x.camino} ⇒ ${x.seccion}`).join('\n'));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('axe:', viol.length ? JSON.stringify(viol) : 'sin violaciones');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.slice(0, 8).join('\n') : 'sin errores');
