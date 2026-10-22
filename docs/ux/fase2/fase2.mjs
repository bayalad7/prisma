// Fase 2: captura la galería del sistema de diseño y los 3 prototipos, revisa axe, reflujo y errores
// de consola (CSP), y comprueba que el panel actual siga funcionando con la hoja nueva.
// Uso: node fase2.mjs <carpeta-salida>
import { chromium } from 'playwright';
import { AxeBuilder } from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

const SALIDA = process.argv[2];
const B = 'https://127.0.0.1:18443/admin/';
const CLAVE = 'contrasena-demo-2026';
fs.mkdirSync(SALIDA, { recursive: true });
const VISTAS = {
  escritorio: { viewport: { width: 1366, height: 768 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};
const informe = { axe: {}, desborde: {}, consola: [] };
const espera = (ms) => new Promise((r) => setTimeout(r, ms));

async function revisar(page, clave) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  informe.axe[clave] = r.violations.map((v) => ({ id: v.id, impacto: v.impact, nodos: v.nodes.length, ej: v.nodes.slice(0, 2).map((n) => n.target.join(' ')) }));
  informe.desborde[clave] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
}
async function cap(page, nombre, completa = true) {
  await page.screenshot({ path: path.join(SALIDA, `${nombre}.png`), fullPage: completa });
}
async function ir(page, url) {
  await page.goto(url); await page.waitForLoadState('networkidle'); await espera(400);
}

const browser = await chromium.launch();
for (const [vp, op] of Object.entries(VISTAS)) {
  for (const tema of ['light', 'dark']) {
    const ctx = await browser.newContext({ ...op, ignoreHTTPSErrors: true, locale: 'es-MX', colorScheme: tema });
    const page = await ctx.newPage();
    page.on('console', (m) => { if (m.type() === 'error' || m.type() === 'warning') informe.consola.push(`${vp}/${tema} ${m.type()}: ${m.text()}`); });
    page.on('pageerror', (e) => informe.consola.push(`${vp}/${tema} pageerror: ${e.message}`));
    const t = tema === 'light' ? 'claro' : 'oscuro';
    const galeria = tema === 'light' || vp === 'escritorio'
      ? ['fundamentos', 'componentes', 'formularios', 'estados', 'contenido'] : ['componentes'];
    for (const s of galeria) {
      await ir(page, `${B}estilos#${s}`);
      if (s === 'formularios') {
        await page.getByRole('button', { name: 'Guardar' }).click(); await espera(200);
      }
      await revisar(page, `estilos-${s}-${vp}-${t}`);
      await cap(page, `estilos-${s}-${vp}-${t}`);
    }
    for (const [nombre, url] of [['inicio', 'proto-inicio'], ['impresora-paso1', 'proto-impresora'], ['impresora-paso2', 'proto-impresora?paso=1'], ['impresora-paso3', 'proto-impresora?paso=2'], ['trabajos', 'proto-trabajos']]) {
      await ir(page, `${B}estilos#${url}`);
      await page.evaluate(() => dispatchEvent(new HashChangeEvent('hashchange'))); await espera(300);
      await revisar(page, `proto-${nombre}-${vp}-${t}`);
      await cap(page, `proto-${nombre}-${vp}-${t}`);
    }
    // Detalle de un trabajo fallido
    await page.getByRole('button', { name: /VTA-1041/ }).first().click();
    await page.waitForSelector('dialog'); await espera(300);
    await revisar(page, `proto-trabajo-fallido-${vp}-${t}`);
    await cap(page, `proto-trabajo-fallido-${vp}-${t}`, false);
    await page.keyboard.press('Escape');
    if (vp === 'celular') {
      await ir(page, `${B}estilos#proto-inicio`);
      await page.getByRole('button', { name: 'Menú' }).click(); await espera(250);
      await cap(page, `proto-menu-${vp}-${t}`, false);
    }
    // El panel actual con la hoja nueva (sin rediseñar): inicio, impresoras y trabajos.
    if (tema === 'light') {
      await ir(page, B);
      if (await page.locator('#contrasena').count()) { await page.fill('#contrasena', CLAVE); await page.getByRole('button', { name: 'Entrar' }).click(); await espera(1200); }
      for (const s of ['inicio', 'impresoras', 'trabajos', 'ayuda']) {
        await ir(page, `${B}#${s}`); await page.evaluate(() => dispatchEvent(new HashChangeEvent('hashchange'))); await espera(900);
        informe.desborde[`panel-${s}-${vp}`] = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
        await cap(page, `panel-actual-${s}-${vp}`);
      }
    }
    await ctx.close();
  }
}
await browser.close();
fs.writeFileSync(path.join(SALIDA, 'informe.json'), JSON.stringify(informe, null, 2));
const viol = Object.entries(informe.axe).filter(([, v]) => v.length);
console.log('violaciones axe:', viol.length ? JSON.stringify(viol, null, 1) : 'ninguna');
console.log('desbordes:', JSON.stringify(Object.entries(informe.desborde).filter(([, v]) => v > 0)));
console.log('consola:', informe.consola.length ? informe.consola.join('\n') : 'sin errores');
// Final del asistente en celular, sin captura de página completa: la barra inferior no debe tapar el botón.
{
  const br = await chromium.launch();
  const ctx = await br.newContext({ ...VISTAS.celular, ignoreHTTPSErrors: true });
  const p = await ctx.newPage();
  await p.goto(`${B}estilos#proto-impresora`); await espera(500);
  await p.evaluate(() => window.scrollTo(0, document.body.scrollHeight)); await espera(300);
  await p.screenshot({ path: path.join(SALIDA, 'proto-impresora-paso1-celular-final.png') });
  const tapado = await p.evaluate(() => {
    const b = [...document.querySelectorAll('button')].find((x) => x.textContent.includes('Continuar'));
    const nav = document.querySelector('.nav-inferior').getBoundingClientRect();
    return b.getBoundingClientRect().bottom > nav.top;
  });
  console.log('Continuar tapado por la barra inferior:', tapado);
  await br.close();
}
