// Compila PRISMA desde el código del repositorio en e2e/.prisma/, una sola vez antes de las
// pruebas: los puentes de prueba (puente.mjs) usan ese binario.
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const aqui = path.dirname(fileURLToPath(import.meta.url));
const salida = path.join(aqui, '.prisma', process.platform === 'win32' ? 'prisma.exe' : 'prisma');
fs.mkdirSync(path.dirname(salida), { recursive: true });
console.log('Compilando PRISMA para las pruebas…');
execFileSync('go', ['build', '-o', salida, './cmd/prisma'], { cwd: path.join(aqui, '..'), stdio: 'inherit', env: { ...process.env, CGO_ENABLED: '0' } });
