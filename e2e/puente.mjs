// Arranca un puente de prueba recién instalado en el puerto indicado: carpeta de datos nueva, sin
// límites de peticiones (las pruebas hacen muchas desde la misma IP) y una impresora de red simulada
// (`prisma dev impresora-virtual`) en el puerto + 1000. Lo usa playwright.config.mjs (webServer);
// al terminar las pruebas, Playwright lo detiene.
//
// Uso: node puente.mjs <puerto>
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const puerto = Number(process.argv[2]);
const aqui = path.dirname(fileURLToPath(import.meta.url));
const binario = path.join(aqui, '.prisma', process.platform === 'win32' ? 'prisma.exe' : 'prisma');
if (!puerto || !fs.existsSync(binario)) {
  console.error('Uso: node puente.mjs <puerto> (antes, node construir.mjs)');
  process.exit(2);
}
// Una carpeta fija por puerto, vaciada al empezar: en Windows, Playwright termina el proceso sin
// avisar y no hay ocasión de borrarla al final.
const datos = path.join(os.tmpdir(), `prisma-e2e-${puerto}`);
fs.rmSync(datos, { recursive: true, force: true });
fs.mkdirSync(datos, { recursive: true });
const config = path.join(datos, 'config.toml');
fs.writeFileSync(config, [
  '[servidor]', 'escuchar = "127.0.0.1"', `puerto = ${puerto}`,
  '[datos]', `ruta = '${datos.replace(/\\/g, '/')}'`,
  '[limites]', 'peticiones_ip_por_minuto = 0', 'peticiones_cliente_por_minuto = 0', '',
].join('\n'));

const hijos = [
  spawn(binario, ['dev', 'impresora-virtual', '--puerto', String(puerto + 1000), '--salida', path.join(datos, 'papel'), '--sin-color'], { stdio: 'ignore' }),
  spawn(binario, ['ejecutar', '--config', config], { stdio: ['ignore', 'inherit', 'inherit'] }),
];
const terminar = () => {
  for (const h of hijos) h.kill();
  setTimeout(() => { fs.rmSync(datos, { recursive: true, force: true }); process.exit(0); }, 500);
};
process.on('SIGINT', terminar);
process.on('SIGTERM', terminar);
for (const h of hijos) h.on('exit', (codigo) => { if (codigo) { console.error(`un proceso del puente terminó con ${codigo}`); terminar(); } });
