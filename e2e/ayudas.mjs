// Ayudas comunes de las pruebas de punta a punta.
import { expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

export const CONTRASENA = 'una frase larga para las pruebas';
export const CONTRASENA_NUEVA = 'otra frase larga para las pruebas';

// Espera a que la pantalla termine de cargar (sin esqueletos ni peticiones pendientes).
export async function listo(page) {
  await page.waitForSelector('.vista .esqueleto', { state: 'detached', timeout: 15_000 }).catch(() => {});
  await page.waitForLoadState('networkidle');
}

// Cambia de pantalla como un enlace del panel, sin recargar.
export async function irA(page, ruta) {
  await page.evaluate((r) => { history.pushState(null, '', r); dispatchEvent(new PopStateEvent('popstate')); }, ruta);
  await listo(page);
}

// Sin violaciones críticas ni serias de WCAG 2.1 A y AA (axe).
export async function sinViolaciones(page, donde) {
  const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  const graves = r.violations.filter((v) => v.impact === 'critical' || v.impact === 'serious')
    .map((v) => `${v.id} (${v.impact}): ${v.nodes.slice(0, 2).map((n) => n.target.join(' ')).join(', ')}`);
  expect(graves, `axe en ${donde}`).toEqual([]);
  // Y nada se sale de la pantalla a lo ancho.
  expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth), `desborde en ${donde}`).toBeLessThanOrEqual(0);
  // Ni un valor aplastado en una lista de datos (con otras fuentes, una etiqueta larga lo dejaba en 0 px).
  const aplastados = await page.evaluate(() => [...document.querySelectorAll('.datos dd')]
    .filter((d) => d.offsetParent && d.textContent.trim() && d.getBoundingClientRect().width < 80).map((d) => d.textContent.trim().slice(0, 30)));
  expect(aplastados, `valores aplastados en ${donde}`).toEqual([]);
}

// Imprime como lo haría una aplicación: POST /v1/imprimir con su origen y su llave.
export async function imprimirComoApp(baseURL, { origen, llave, impresora, referencia, operaciones }) {
  const cabeceras = { 'Content-Type': 'application/json' };
  if (origen) cabeceras.Origin = origen;
  if (llave) cabeceras.Authorization = `Bearer ${llave}`;
  const r = await fetch(`${baseURL}/v1/imprimir`, { method: 'POST', headers: cabeceras, body: JSON.stringify({
    version: '1', id: crypto.randomUUID(), impresora, referencia,
    operaciones: operaciones || [{ tipo: 'texto', valor: 'Ticket de prueba e2e' }, { tipo: 'cortar' }],
  }) });
  return { estado: r.status, datos: await r.json() };
}
