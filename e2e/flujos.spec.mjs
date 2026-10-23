// Flujos a)–g) de docs/ux/ANALISIS-PANEL-V1.md, de un puente recién instalado hasta un integrador
// que prueba su aplicación sin impresora, en orden y con axe en cada pantalla. Cada proyecto
// (escritorio y celular) tiene su propio puente.
import { test, expect } from '@playwright/test';
import { generateKeyPairSync } from 'node:crypto';
import { CONTRASENA, CONTRASENA_NUEVA, listo, irA, sinViolaciones, imprimirComoApp } from './ayudas.mjs';

test.describe.configure({ mode: 'serial' });

let page;
let baseURL;
let llaveApp = '';
const ORIGEN_APP = 'https://pos.ejemplo.test';

test.beforeAll(async ({ browser }, info) => {
  baseURL = info.project.use.baseURL;
  const ctx = await browser.newContext({ ...info.project.use });
  page = await ctx.newPage();
  page.on('pageerror', (e) => { throw new Error(`error en la página: ${e.message}`); });
});
test.afterAll(async () => { await page.context().close(); });

// La tarjeta de una impresora, por su título (el alias).
const tarjetaImpresora = (alias) => page.locator('.lista > .tarjeta').filter({ has: page.getByRole('heading', { name: alias, exact: true }) });

test('a) primer arranque: crear la contraseña y entrar', async () => {
  await page.goto('/admin/');
  await expect(page.getByRole('heading', { name: 'Bienvenido a PRISMA' })).toBeVisible();
  await sinViolaciones(page, 'configuración inicial');
  await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
  await expect(page.locator('#nueva')).toHaveAttribute('aria-invalid', 'true');
  await page.fill('#nueva', CONTRASENA);
  await page.fill('#repetir', CONTRASENA);
  await page.getByRole('button', { name: 'Crear contraseña y entrar' }).click();
  await listo(page);
  await expect(page.getByText('Primeros pasos')).toBeVisible();
  await expect(page.getByText(/1 de \d pasos completados/)).toBeVisible();
  await sinViolaciones(page, 'Inicio con primeros pasos');
});

test('b) primera impresora: asistente con una impresora de red, prueba y acentos', async ({}, info) => {
  const simulada = `127.0.0.1:${new URL(baseURL).port * 1 + 1000}`;
  await irA(page, '/admin/impresoras');
  await page.getByRole('button', { name: 'Agregar impresora' }).first().click();
  await sinViolaciones(page, 'asistente: tipo');
  await page.locator('label.opcion', { hasText: 'De tickets de 80 mm' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click();
  await page.locator('summary', { hasText: '¿No aparece?' }).click();
  await page.locator('#as-destino').fill(simulada);
  await page.locator('#as-nombre').fill('Caja');
  await page.getByRole('button', { name: 'Probar conexión' }).click();
  await expect(page.getByText('La impresora respondió: está lista.')).toBeVisible();
  await sinViolaciones(page, 'asistente: conexión');
  await page.getByRole('button', { name: 'Guardar y probar' }).click();
  await expect(page.getByText('¿Salió el ticket?')).toBeVisible({ timeout: 20_000 });
  await page.getByRole('button', { name: 'Sí, salió' }).click();
  await page.getByRole('button', { name: 'Sí, se leen bien' }).click();
  await expect(page.getByText(/Listo: «caja» ya puede imprimir/)).toBeVisible();
  await sinViolaciones(page, 'asistente: listo');
  void info;
});

test('b) segundo camino del asistente: una impresora virtual', async () => {
  await irA(page, '/admin/impresoras');
  await page.getByRole('button', { name: 'Agregar impresora' }).first().click();
  await page.locator('label.opcion', { hasText: 'Virtual, para probar' }).click();
  await page.getByRole('button', { name: 'Continuar' }).click();
  await page.getByRole('button', { name: 'Crear y probar' }).click();
  await expect(page.getByText(/Listo: «prueba» ya puede imprimir/)).toBeVisible({ timeout: 20_000 });
  await irA(page, '/admin/impresoras');
  await expect(tarjetaImpresora('prueba').getByText('Virtual', { exact: true })).toBeVisible();
  await sinViolaciones(page, 'Impresoras');
});

test('c) aplicación de nivel 2: llave, código para integrar y primer ticket', async () => {
  await irA(page, '/admin/aplicaciones');
  await page.getByRole('button', { name: 'Agregar aplicación' }).first().click();
  await page.locator('#cli-nombre').fill('Punto de venta');
  await page.locator('#cli-origen').fill(ORIGEN_APP);
  await page.getByRole('button', { name: 'Agregar', exact: true }).click();
  await sinViolaciones(page, 'nueva aplicación');
  await page.getByRole('button', { name: 'Crear aplicación' }).click();
  const dialogo = page.locator('dialog[open]');
  await expect(dialogo.getByText('Cópiala o descárgala ahora')).toBeVisible();
  llaveApp = (await dialogo.locator('.llave-caja code').innerText()).trim();
  expect(llaveApp).toMatch(/^pk_/);
  await expect(dialogo.locator('pre').first()).toContainText(llaveApp);
  await sinViolaciones(page, 'llave de la aplicación');
  await dialogo.getByRole('button', { name: 'Ya la guardé' }).click();
  await listo(page);
  // «Probar»: la aplicación imprime de verdad y el ticket llega a la impresora virtual.
  const r = await imprimirComoApp(baseURL, { origen: ORIGEN_APP, llave: llaveApp, impresora: 'prueba', referencia: 'E2E-C' });
  expect(r, JSON.stringify(r.datos)).toMatchObject({ estado: 200, datos: { ok: true } });
  await irA(page, '/admin/pruebas/rollo/prueba');
  await expect(page.locator('.ticket-rollo', { hasText: 'Punto de venta · E2E-C' })).toBeVisible({ timeout: 8_000 });
});

test('c) aplicación de nivel 3 sin llave: el panel da el id con que su servidor firma', async () => {
  const { publicKey, privateKey } = generateKeyPairSync('ed25519');
  await irA(page, '/admin/aplicaciones');
  await page.getByRole('button', { name: 'Agregar aplicación' }).first().click();
  await page.locator('#cli-nombre').fill('Pedidos en línea');
  await page.locator('label.opcion', { hasText: 'Firma desde su servidor' }).click();
  await page.locator('#cli-publica').fill(publicKey.export({ type: 'spki', format: 'pem' }));
  await page.getByRole('button', { name: 'Crear aplicación' }).click();
  const dialogo = page.locator('dialog[open]');
  await expect(dialogo.getByText('su backend necesita este id')).toBeVisible();
  const id = (await dialogo.locator('.llave-caja code').innerText()).trim();
  expect(id).toMatch(/^cli_[A-Za-z0-9]{16}$/);
  // El primer ejemplo es la firma en el servidor, con el id real.
  await expect(dialogo.locator('pre').first()).toContainText(`clienteId: "${id}"`);
  await sinViolaciones(page, 'id de una aplicación de nivel 3');
  await dialogo.getByRole('button', { name: 'Entendido' }).click();
  await listo(page);
  await expect(page.locator('.tarjeta', { hasText: 'Pedidos en línea' }).getByText(id, { exact: true })).toBeVisible();

  // Su servidor firma con ese id y el puente imprime, sin origen ni llave.
  const { firmar } = await import('../sdk/firmadores/node/firmador.mjs');
  const firmado = firmar({ version: '1', impresora: 'prueba', referencia: 'E2E-FIRMA',
    operaciones: [{ tipo: 'texto', valor: 'Firmado' }, { tipo: 'cortar' }] }, { clienteId: id, llavePrivada: privateKey });
  const r = await fetch(`${baseURL}/v1/imprimir`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(firmado) });
  expect(r.status, await r.clone().text()).toBe(200);
});

test('d) una página desconocida pide permiso y se aprueba', async () => {
  const origen = 'https://menu.ejemplo.test';
  const r = await imprimirComoApp(baseURL, { origen, impresora: 'prueba' });
  expect(r.datos.codigo).toBe('ORIGEN_NO_AUTORIZADO');
  await irA(page, '/admin/solicitudes');
  const tarjeta = page.locator('.tarjeta', { hasText: origen });
  await expect(tarjeta).toBeVisible();
  await sinViolaciones(page, 'Solicitudes');
  await tarjeta.getByRole('button', { name: 'Aprobar' }).click();
  const dialogo = page.locator('dialog[open]');
  await expect(dialogo.locator('#apr-nombre')).toHaveValue('menu.ejemplo.test');
  await sinViolaciones(page, 'aprobar solicitud');
  await dialogo.getByRole('button', { name: 'Aprobar' }).click();
  await expect(page.locator('dialog[open]').getByText('Cópiala o descárgala ahora')).toBeVisible();
  await page.locator('dialog[open]').getByRole('button', { name: 'Ya la guardé' }).click();
  await irA(page, '/admin/aplicaciones');
  await expect(page.locator('.tarjeta', { hasText: 'menu.ejemplo.test' }).first()).toBeVisible();
});

test('e) un trabajo falla: qué pasó, por qué y qué hacer', async () => {
  // La impresora virtual simula estar fuera de línea.
  await irA(page, '/admin/impresoras');
  await tarjetaImpresora('prueba').getByRole('button', { name: 'Editar' }).click();
  await page.locator('#simulacion').selectOption('fuera_de_linea');
  await page.getByRole('button', { name: 'Guardar cambios' }).click();
  await listo(page);
  const r = await imprimirComoApp(baseURL, { origen: ORIGEN_APP, llave: llaveApp, impresora: 'prueba', referencia: 'E2E-FALLA' });
  expect(r.datos.codigo).toBe('IMPRESORA_INALCANZABLE');
  await irA(page, '/admin/trabajos?estado=fallido');
  await page.locator('tbody tr', { hasText: 'E2E-FALLA' }).locator('.fila-abrir').click();
  const dialogo = page.locator('dialog[open]');
  await expect(dialogo.getByText('Qué pasó')).toBeVisible();
  await expect(dialogo.getByText('Qué hacer')).toBeVisible();
  await expect(dialogo.getByRole('button', { name: 'Probar la impresora' })).toBeVisible();
  await sinViolaciones(page, 'detalle de un trabajo fallido');
  await dialogo.getByRole('button', { name: 'Leer en el manual' }).click();
  await expect(page.locator('dialog.panel-ayuda[open] h2').first()).toHaveText('La impresora no imprime');
  await page.keyboard.press('Escape');
  await page.keyboard.press('Escape');
  // Vuelve a la normalidad para el flujo g).
  await irA(page, '/admin/impresoras');
  await tarjetaImpresora('prueba').getByRole('button', { name: 'Editar' }).click();
  await page.locator('#simulacion').selectOption('normal');
  await page.getByRole('button', { name: 'Guardar cambios' }).click();
  await listo(page);
});

test('f) contraseña, certificado y respaldo', async () => {
  await irA(page, '/admin/ajustes?pestana=seguridad');
  await page.locator('#aj-actual').fill(CONTRASENA);
  await page.locator('#aj-nueva').fill(CONTRASENA_NUEVA);
  await page.locator('#aj-repetir').fill(CONTRASENA_NUEVA);
  await page.getByRole('button', { name: 'Cambiar contraseña' }).click();
  await expect(page.getByText(/Contraseña cambiada/)).toBeVisible();
  await sinViolaciones(page, 'Ajustes: seguridad');
  await page.getByRole('tab', { name: 'HTTPS' }).click();
  await page.getByRole('button', { name: 'Regenerar certificado' }).click();
  await expect(page.locator('dialog[open]').getByText(/tendrá que volver a confiar/)).toBeVisible();
  await page.locator('dialog[open]').getByRole('button', { name: 'Cancelar' }).click();
  await page.getByRole('tab', { name: 'Respaldo' }).click();
  const [descarga] = await Promise.all([page.waitForEvent('download'), page.getByRole('link', { name: 'Descargar respaldo' }).click()]);
  expect(descarga.suggestedFilename()).toMatch(/\.db$/);
  await sinViolaciones(page, 'Ajustes: respaldo');
});

test('g) un integrador prueba su aplicación sin impresora', async () => {
  // Probar impresión: plantilla, vista previa y envío a la virtual.
  await irA(page, '/admin/pruebas?impresora=prueba');
  await expect(page.locator('.previa-prueba img')).toBeVisible();
  await sinViolaciones(page, 'Probar impresión');
  await page.getByRole('button', { name: 'Enviar a «prueba»' }).click();
  await expect(page.getByText('«prueba» recibió el ticket.')).toBeVisible({ timeout: 15_000 });
  await page.getByRole('button', { name: 'Copiar como código' }).click();
  await expect(page.locator('dialog[open] pre').first()).toContainText('prisma.imprimir("prueba"');
  await page.keyboard.press('Escape');

  // Su aplicación con el SDK, como en un backend: el ticket aparece solo en el rollo en vivo.
  await irA(page, '/admin/pruebas/rollo/prueba');
  await sinViolaciones(page, 'rollo en vivo');
  const { Prisma } = await import('../sdk/js/dist/prisma.mjs');
  const prisma = new Prisma({ url: baseURL, llave: llaveApp });
  const r = await prisma.imprimir('prueba', (doc) => doc.texto('Hola desde el SDK').cortar(), { referencia: 'E2E-SDK' });
  expect(r.ok).toBe(true);
  await expect(page.locator('.ticket-rollo', { hasText: 'E2E-SDK' })).toBeVisible({ timeout: 8_000 });

  // ¿Por qué mi app no puede imprimir? Sin llave, el comprobador lo dice.
  await irA(page, '/admin/pruebas/permisos');
  await page.locator('#pe-app').selectOption({ label: 'Punto de venta' });
  const resumen = page.locator('.tarjeta-previa .nota-caja').first();
  await expect(resumen).toContainText('Puede imprimir');
  await page.locator('label[for="pe-llave"]').click();
  await expect(resumen).toContainText('La API respondería 401 LLAVE_REQUERIDA');
  await sinViolaciones(page, 'comprobador de permisos');
});
