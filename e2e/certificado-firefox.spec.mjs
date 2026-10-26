// Firefox real contra el certificado del puente (ADR 0021): no acepta que una autoridad sea el
// certificado del sitio (MOZILLA_PKIX_ERROR_CA_CERT_USED_AS_END_ENTITY, el fallo de 0.3.0). Con la
// autoridad local en la base NSS del perfil, como hace «prisma tls confiar» con la del sistema, el
// puente debe abrir sin advertencia; sin ella, no. Necesita certutil (libnss3-tools): corre en el
// CI de Linux y se salta donde no está.
import { test, expect, firefox } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

// El certutil de NSS (en Windows, «certutil» es otra herramienta).
const hayCertutil = process.platform === 'linux' && (() => {
  try { execFileSync('certutil', ['-H'], { stdio: 'ignore' }); return true; } catch (e) { return e.code !== 'ENOENT'; }
})();

test('Firefox abre el puente sin advertencia solo si confía en su autoridad local', async ({}, info) => {
  test.skip(!hayCertutil, 'falta certutil (libnss3-tools)');
  const puerto = new URL(info.project.use.baseURL).port;
  const autoridad = path.join(os.tmpdir(), `prisma-e2e-${puerto}`, 'tls', 'autoridad.pem');
  expect(fs.existsSync(autoridad), 'el puente debe tener autoridad local').toBe(true);
  const perfil = fs.mkdtempSync(path.join(os.tmpdir(), 'prisma-firefox-'));
  execFileSync('certutil', ['-N', '-d', `sql:${perfil}`, '--empty-password']);
  const abrir = async () => {
    const ctx = await firefox.launchPersistentContext(perfil);
    try {
      const r = await ctx.pages()[0].goto(`https://localhost:${puerto}/v1/salud`, { timeout: 15_000 });
      return `abrió ${r.status()}`;
    } catch (e) {
      return (e.message.match(/(SEC_ERROR_\w+|MOZILLA_PKIX_ERROR_\w+|SSL_ERROR_\w+)/) || [e.message])[0];
    } finally {
      await ctx.close();
    }
  };
  expect(await abrir(), 'sin la autoridad, Firefox no debe confiar').toBe('SEC_ERROR_UNKNOWN_ISSUER');
  execFileSync('certutil', ['-A', '-n', 'PRISMA autoridad local', '-t', 'C,,', '-i', autoridad, '-d', `sql:${perfil}`]);
  expect(await abrir()).toBe('abrió 200');
  fs.rmSync(perfil, { recursive: true, force: true });
});
