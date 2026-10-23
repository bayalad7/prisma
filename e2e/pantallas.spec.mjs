// Todas las pantallas del panel, en claro y en oscuro, con axe (sin violaciones críticas ni serias) y
// sin desbordes. Corre después de flujos.spec.mjs (el puente ya tiene impresoras, aplicaciones y
// trabajos). Con CAPTURAS=<carpeta> guarda una captura de cada una en tema claro (docs/ux/despues/).
import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { CONTRASENA_NUEVA, listo, irA, sinViolaciones, imprimirComoApp } from './ayudas.mjs';

test.describe.configure({ mode: 'serial' });

let page;
let baseURL;
let proyecto;
const CAPTURAS = process.env.CAPTURAS;

test.beforeAll(async ({ browser }, info) => {
  baseURL = info.project.use.baseURL;
  proyecto = info.project.name.replace('-pantallas', '');
  page = await (await browser.newContext({ ...info.project.use })).newPage();
  page.on('pageerror', (e) => { throw new Error(`error en la página: ${e.message}`); });
  await page.goto('/admin/');
  await page.fill('#contrasena', CONTRASENA_NUEVA);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await listo(page);
  // Una página nueva pide permiso, para ver Solicitudes con algo.
  await imprimirComoApp(baseURL, { origen: 'http://192.168.1.40:5173', impresora: 'prueba' });
});
test.afterAll(async () => { await page.context().close(); });

const cerrarDialogos = async () => {
  for (let k = 0; k < 3 && await page.locator('dialog[open]').count(); k++) await page.keyboard.press('Escape');
};

// [nombre (como en docs/ux/antes/), ruta, qué hacer antes de revisar]
const PANTALLAS = [
  ['12-inicio', '/admin/'],
  ['20-impresoras', '/admin/impresoras'],
  ['22-diagnostico', '/admin/impresoras', async () => {
    await page.locator('.lista > .tarjeta').filter({ has: page.getByRole('heading', { name: 'caja', exact: true }) })
      .getByRole('button', { name: 'Ticket de prueba' }).click();
    await expect(page.locator('dialog[open] img')).toBeVisible({ timeout: 15_000 });
  }],
  ['25-asistente-paso1', '/admin/impresoras', () => page.getByRole('button', { name: 'Agregar impresora' }).first().click()],
  ['30-impresora-editar', '/admin/impresoras', () => page.locator('.lista > .tarjeta').filter({ has: page.getByRole('heading', { name: 'prueba', exact: true }) })
    .getByRole('button', { name: 'Editar' }).click()],
  ['40-aplicaciones', '/admin/aplicaciones'],
  ['41-codigo-integracion', '/admin/aplicaciones', () => page.locator('.tarjeta', { hasText: 'Punto de venta' }).first()
    .getByRole('button', { name: 'Código para integrar' }).click()],
  ['42-aplicacion-nueva', '/admin/aplicaciones', () => page.getByRole('button', { name: 'Agregar aplicación' }).first().click()],
  ['50-solicitudes', '/admin/solicitudes'],
  ['51-aprobar-origen', '/admin/solicitudes', () => page.locator('.tarjeta', { hasText: '192.168.1.40' }).getByRole('button', { name: 'Aprobar' }).click()],
  ['60-trabajos', '/admin/trabajos'],
  ['61-trabajos-fallidos', '/admin/trabajos?estado=fallido'],
  ['62-trabajo-detalle', '/admin/trabajos?estado=fallido', () => page.locator('tbody tr .fila-abrir').first().click()],
  ['70-imagenes', '/admin/imagenes'],
  ['80-ajustes-general', '/admin/ajustes'],
  ['81-ajustes-seguridad', '/admin/ajustes?pestana=seguridad'],
  ['82-ajustes-https', '/admin/ajustes?pestana=https'],
  ['83-ajustes-respaldo', '/admin/ajustes?pestana=respaldo'],
  ['84-probar-impresion', '/admin/pruebas?impresora=prueba', () => expect(page.locator('.previa-prueba img')).toBeVisible()],
  ['85-probar-constructor', '/admin/pruebas?impresora=prueba&modo=constructor', () => expect(page.locator('details.operacion').first()).toBeVisible()],
  ['86-probar-json', '/admin/pruebas?impresora=prueba&modo=json'],
  ['87-permisos', '/admin/pruebas/permisos', () => expect(page.locator('.paso-permiso').first()).toBeVisible()],
  ['88-rollo', '/admin/pruebas/rollo/prueba', () => expect(page.locator('.ticket-rollo').first()).toBeVisible()],
  ['90-ayuda', '/admin/ayuda'],
  ['91-ayuda-busqueda', '/admin/ayuda?q=acentos', () => expect(page.locator('.resultados-manual li').first()).toBeVisible()],
  ['92-ayuda-documento', '/admin/ayuda/administracion/primeros-pasos.md'],
  ['93-no-imprime', '/admin/ayuda/no-imprime'],
  ['94-galeria', '/admin/ayuda/diagramas'],
  ['95-diagrama', '/admin/ayuda/diagramas/07-orden-autorizacion.md', () => expect(page.locator('.visor-lienzo img')).toBeVisible()],
  ['96-diagrama-grande', '/admin/ayuda/diagramas/29-impresora-virtual.md', () => page.getByRole('button', { name: 'Ver en grande' }).click()],
  ['97-panel-ayuda', '/admin/impresoras', async () => {
    await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).click();
    await expect(page.locator('dialog.panel-ayuda[open]')).toBeVisible();
  }],
];

for (const tema of ['claro', 'oscuro']) {
  test(`todas las pantallas en tema ${tema}`, async () => {
    await page.emulateMedia({ colorScheme: tema === 'oscuro' ? 'dark' : 'light' });
    for (const [nombre, ruta, antes] of PANTALLAS) {
      await cerrarDialogos();
      await irA(page, ruta);
      if (antes) await antes();
      await page.waitForTimeout(300);
      await sinViolaciones(page, `${nombre} (${tema})`);
      if (CAPTURAS && tema === 'claro') {
        fs.mkdirSync(CAPTURAS, { recursive: true });
        // En escritorio, la página completa; en celular, lo que se ve (las páginas largas pesarían mucho).
        await page.screenshot({ path: path.join(CAPTURAS, `${nombre}-${proyecto}.png`),
          fullPage: proyecto === 'escritorio' && !(await page.locator('dialog[open]').count()) });
      }
    }
    await cerrarDialogos();
  });
}

test('pantallas de acceso: entrar, contraseña equivocada y sin conexión', async ({ browser }, info) => {
  const otra = await (await browser.newContext({ ...info.project.use })).newPage();
  await otra.goto('/admin/');
  await expect(otra.getByRole('heading', { name: 'Entrar al panel' })).toBeVisible();
  await sinViolaciones(otra, 'entrar');
  if (CAPTURAS) await otra.screenshot({ path: path.join(CAPTURAS, `10-entrar-${proyecto}.png`) });
  await otra.fill('#contrasena', 'no-es-la-contrasena');
  await otra.getByRole('button', { name: 'Entrar', exact: true }).click();
  await expect(otra.getByText('La contraseña no es correcta.')).toBeVisible();
  await sinViolaciones(otra, 'entrar con error');
  if (CAPTURAS) await otra.screenshot({ path: path.join(CAPTURAS, `11-entrar-error-${proyecto}.png`) });
  await otra.route('**/admin/api/sesion', (r) => r.abort());
  await otra.reload();
  await expect(otra.getByRole('button', { name: 'Volver a intentar' })).toBeVisible();
  await sinViolaciones(otra, 'sin conexión');
  if (CAPTURAS) await otra.screenshot({ path: path.join(CAPTURAS, `98-sin-conexion-${proyecto}.png`) });
  await otra.context().close();
});
