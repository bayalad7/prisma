// Usuario nuevo (Fase 7 de 0.2.0): de un puente recién instalado al primer ticket y a la primera
// aplicación autorizada, por el camino que sugiere «Primeros pasos», contando clics y campos y
// midiendo el tiempo (meta: menos de 5 minutos). Después reinicia el puente y comprueba que los
// tickets de la impresora virtual no sobreviven: el rollo vuelve vacío, el historial se conserva y
// el contenido del ticket no está en ningún archivo de datos.
//
// Maneja su propio puente (puerto 18603), porque tiene que detenerlo y volver a arrancarlo.
import { test, expect } from '@playwright/test';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { CONTRASENA, listo, irA, imprimirComoApp } from './ayudas.mjs';

test.describe.configure({ mode: 'serial' });

const aqui = path.dirname(fileURLToPath(import.meta.url));
const binario = path.join(aqui, '.prisma', process.platform === 'win32' ? 'prisma.exe' : 'prisma');
const PUERTO = 18603;
const datos = path.join(os.tmpdir(), `prisma-e2e-${PUERTO}`);
const config = path.join(datos, 'config.toml');
const ORIGEN_APP = 'https://tienda.ejemplo.test';
const SECRETO = 'Ticket de la señora Peña: 2 tamales';

let puente;
let page;
let llave = '';
const cuenta = { clics: 0, campos: 0 };
const tiempos = {};

async function arrancar() {
  puente = spawn(binario, ['ejecutar', '--config', config], { stdio: 'ignore' });
  for (let k = 0; k < 100; k++) {
    try {
      if ((await fetch(`https://127.0.0.1:${PUERTO}/v1/salud`)).ok) return;
    } catch { /* todavía arranca */ }
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error('el puente no arrancó');
}

async function detener() {
  const salio = new Promise((r) => puente.once('exit', r));
  puente.kill();
  await salio;
}

// Lo que hace la persona, contado.
const clic = async (loc) => { cuenta.clics++; await loc.click(); };
const escribir = async (loc, texto) => { cuenta.campos++; await loc.fill(texto); };

test.beforeAll(async ({ browser }, info) => {
  test.skip(!fs.existsSync(binario), 'antes, node construir.mjs');
  fs.rmSync(datos, { recursive: true, force: true });
  fs.mkdirSync(datos, { recursive: true });
  fs.writeFileSync(config, [
    '[servidor]', 'escuchar = "127.0.0.1"', `puerto = ${PUERTO}`,
    '[datos]', `ruta = '${datos.replace(/\\/g, '/')}'`, '',
  ].join('\n'));
  await arrancar();
  page = await (await browser.newContext({ ...info.project.use })).newPage();
  page.on('pageerror', (e) => { throw new Error(`error en la página: ${e.message}`); });
});

test.afterAll(async () => {
  await page?.context().close();
  if (puente && puente.exitCode === null) await detener();
  fs.rmSync(datos, { recursive: true, force: true });
});

test('de recién instalado al primer ticket y a la primera aplicación en menos de 5 minutos', async () => {
  const inicio = Date.now();
  await page.goto('/admin/');
  await escribir(page.locator('#nueva'), CONTRASENA);
  await escribir(page.locator('#repetir'), CONTRASENA);
  await clic(page.getByRole('button', { name: 'Crear contraseña y entrar' }));
  await listo(page);
  await expect(page.getByText('Primeros pasos')).toBeVisible();

  // Primer ticket: una impresora virtual desde «Primeros pasos» y su ticket de prueba.
  await clic(page.getByRole('button', { name: 'Crear impresora virtual de prueba' }).first());
  await listo(page);
  // Lleva a Impresoras, con la tarjeta de la impresora nueva.
  await clic(page.getByRole('button', { name: 'Ticket de prueba' }));
  await expect(page.locator('dialog[open] img')).toBeVisible({ timeout: 15_000 });
  tiempos.primerTicket = Date.now() - inicio;
  await clic(page.locator('dialog[open]').getByRole('button', { name: 'Cerrar' }).first());

  // Primera aplicación: de vuelta en Inicio, «Autorizar», la llave y el primer ticket de la app.
  await clic(page.getByRole('link', { name: 'Inicio' }));
  await listo(page);
  await clic(page.getByRole('link', { name: 'Autorizar' }));
  await listo(page);
  await clic(page.getByRole('button', { name: 'Agregar aplicación' }).first());
  await escribir(page.locator('#cli-nombre'), 'Tienda en línea');
  await escribir(page.locator('#cli-origen'), ORIGEN_APP);
  await clic(page.getByRole('button', { name: 'Agregar', exact: true }));
  await clic(page.getByRole('button', { name: 'Crear aplicación' }));
  const dialogo = page.locator('dialog[open]');
  llave = (await dialogo.locator('.llave-caja code').innerText()).trim();
  await clic(dialogo.getByRole('button', { name: 'Ya la guardé' }));
  await listo(page);
  const alias = (await (await fetch(`https://127.0.0.1:${PUERTO}/v1/impresoras`, {
    headers: { Origin: ORIGEN_APP, Authorization: `Bearer ${llave}` } })).json())[0].alias;
  const r = await imprimirComoApp(`https://127.0.0.1:${PUERTO}`, { origen: ORIGEN_APP, llave, impresora: alias, referencia: 'USUARIO-NUEVO',
    operaciones: [{ tipo: 'texto', valor: SECRETO }, { tipo: 'cortar' }] });
  expect(r, JSON.stringify(r.datos)).toMatchObject({ estado: 200, datos: { ok: true } });
  await irA(page, `/admin/pruebas/rollo/${alias}`);
  await expect(page.locator('.ticket-rollo', { hasText: 'USUARIO-NUEVO' })).toBeVisible({ timeout: 8_000 });
  tiempos.aplicacion = Date.now() - inicio;
  tiempos.alias = alias;

  test.info().annotations.push({ type: 'usuario-nuevo',
    description: `primer ticket en ${(tiempos.primerTicket / 1000).toFixed(1)} s; aplicación imprimiendo en ` +
      `${(tiempos.aplicacion / 1000).toFixed(1)} s; ${cuenta.clics} clics y ${cuenta.campos} campos` });
  console.log(`usuario nuevo: ${test.info().annotations.at(-1).description}`);
  expect(tiempos.aplicacion).toBeLessThan(5 * 60_000);
});

test('al reiniciar el puente, los tickets de la impresora virtual se borran y el historial queda', async () => {
  await detener();
  await arrancar();
  await page.goto('/admin/');
  await page.fill('#contrasena', CONTRASENA);
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await listo(page);
  await irA(page, `/admin/pruebas/rollo/${tiempos.alias}`);
  await expect(page.getByText('Todavía no llegó ningún ticket')).toBeVisible();
  await irA(page, '/admin/trabajos');
  await expect(page.locator('tbody tr', { hasText: 'USUARIO-NUEVO' })).toBeVisible();
  // Ni la base ni la bitácora ni ningún otro archivo de datos tiene el contenido del ticket.
  const archivos = fs.readdirSync(datos, { recursive: true }).map((a) => path.join(datos, a)).filter((a) => fs.statSync(a).isFile());
  expect(archivos.some((a) => path.basename(a) === 'prisma.db')).toBe(true);
  for (const a of archivos) {
    expect(fs.readFileSync(a).includes(Buffer.from('señora Peña')), `contenido del ticket en ${a}`).toBe(false);
  }
});
