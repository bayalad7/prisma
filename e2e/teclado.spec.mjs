// Navegación solo con teclado (escritorio): saltar al contenido, el menú con Tab, los atajos, el
// asistente sin ratón y los diálogos, que se cierran con Escape y devuelven el foco.
import { test, expect } from '@playwright/test';
import { CONTRASENA_NUEVA, listo } from './ayudas.mjs';

test.describe.configure({ mode: 'serial' });

let page;
test.beforeAll(async ({ browser }, info) => {
  page = await (await browser.newContext({ ...info.project.use })).newPage();
  await page.goto('/admin/');
  // Entrar también con el teclado.
  await page.locator('#contrasena').focus();
  await page.keyboard.type(CONTRASENA_NUEVA);
  await page.keyboard.press('Enter');
  await listo(page);
});
test.afterAll(async () => { await page.context().close(); });

const enfocado = () => page.evaluate(() => {
  const e = document.activeElement;
  return { tag: e.tagName, texto: (e.getAttribute('aria-label') || e.textContent || '').trim().slice(0, 60), id: e.id, href: e.getAttribute('href') };
});

test('lo primero que se enfoca es «Saltar al contenido», y lleva al contenido', async () => {
  // Al cambiar de pantalla el foco va a su título (para el lector de pantalla); el enlace para
  // saltar es el primer elemento enfocable de la página, el que alcanza Tab desde el principio.
  const primero = await page.evaluate(() => {
    const e = document.querySelector('a[href], button:not([disabled]), input:not([disabled]), select, textarea, [tabindex]:not([tabindex="-1"])');
    return (e.textContent || '').trim();
  });
  expect(primero).toMatch(/Saltar al contenido/);
  await page.getByRole('link', { name: /Saltar al contenido/ }).focus();
  await page.keyboard.press('Enter');
  expect(await page.evaluate(() => !!document.activeElement.closest('main'))).toBe(true);
});

test('el menú se recorre con Tab y se abre una sección con Enter', async () => {
  await page.locator('nav a[href="/admin/"]').first().focus();
  for (let k = 0; k < 15; k++) {
    if ((await enfocado()).href === '/admin/impresoras') break;
    await page.keyboard.press('Tab');
  }
  expect((await enfocado()).href).toBe('/admin/impresoras');
  await page.keyboard.press('Enter');
  await listo(page);
  expect(page.url()).toMatch(/\/admin\/impresoras$/);
  // Al cambiar de pantalla, el foco va a su título.
  expect((await enfocado()).tag).toBe('H1');
});

test('los atajos g y una letra cambian de sección', async () => {
  await page.keyboard.press('g');
  await page.keyboard.press('t');
  await listo(page);
  expect(page.url()).toMatch(/\/admin\/trabajos$/);
  await page.keyboard.press('g');
  await page.keyboard.press('i');
  await listo(page);
  expect(page.url()).toMatch(/\/admin\/$/);
});

test('el asistente de impresoras se usa sin ratón', async () => {
  await page.goto('/admin/impresoras');
  await listo(page);
  await page.getByRole('button', { name: 'Agregar impresora' }).first().focus();
  await page.keyboard.press('Enter');
  // Elegir «Virtual» con el teclado: Tab hasta las opciones y las flechas.
  const virtual = page.locator('label.opcion', { hasText: 'Virtual, para probar' }).locator('input');
  await virtual.focus();
  await page.keyboard.press('Space');
  await expect(virtual).toBeChecked();
  await page.getByRole('button', { name: 'Continuar' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#as-nombre')).toBeVisible();
  // «Cancelar» es un enlace que vuelve a Impresoras.
  await page.getByRole('link', { name: 'Cancelar' }).focus();
  await page.keyboard.press('Enter');
  await listo(page);
  await expect(page.getByRole('heading', { name: 'Impresoras', exact: true })).toBeVisible();
});

test('un diálogo se cierra con Escape y devuelve el foco', async () => {
  await page.goto('/admin/trabajos');
  await listo(page);
  const abrir = page.locator('tbody tr .fila-abrir').first();
  await abrir.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('dialog[open]')).toBeVisible();
  expect(await page.evaluate(() => !!document.activeElement.closest('dialog[open]'))).toBe(true);
  await page.keyboard.press('Escape');
  await expect(page.locator('dialog[open]')).toHaveCount(0);
  expect(await page.evaluate(() => document.activeElement.classList.contains('fila-abrir'))).toBe(true);
});

test('el panel de ayuda y el visor de diagramas, con teclado', async () => {
  await page.getByRole('button', { name: 'Ayuda de esta pantalla' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('dialog.panel-ayuda[open]')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.locator('dialog.panel-ayuda[open]')).toHaveCount(0);
  expect((await enfocado()).texto).toBe('Ayuda de esta pantalla');
  await page.goto('/admin/ayuda/diagramas/07-orden-autorizacion.md');
  await listo(page);
  await page.locator('.visor-lienzo').focus();
  await page.keyboard.press('+');
  expect(await page.locator('.visor-lienzo img').evaluate((i) => i.style.transform)).toContain('scale(1.25)');
  await page.keyboard.press('0');
});
