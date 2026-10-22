<?php

// POST firmar.php {"productos": [{"clave": "taco", "cantidad": 2}, …]}
//
// El backend arma el ticket con SUS datos (precios del catálogo, folio, fecha) y lo firma. El
// navegador solo elige productos y cantidades: si alguien altera el documento firmado antes de
// enviarlo al puente, la firma deja de verificar y el puente lo rechaza con FIRMA_INVALIDA.
declare(strict_types=1);

// En tu proyecto, copia PrismaFirmador.php (sdk/firmadores/php) a tu código.
require __DIR__ . '/../../../../sdk/firmadores/php/PrismaFirmador.php';
$config = require __DIR__ . '/config.php';

header('Content-Type: application/json; charset=utf-8');

function responderError(int $estado, string $mensaje): never
{
    http_response_code($estado);
    echo json_encode(['ok' => false, 'mensaje' => $mensaje], JSON_UNESCAPED_UNICODE);
    exit;
}

// En una aplicación real: exige aquí la sesión del usuario y un token CSRF.
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    responderError(405, 'Usa POST.');
}

// Catálogo del servidor: el navegador no puede cambiar los precios.
const CATALOGO = [
    'taco' => ['Taco al pastor', 3000],
    'quesadilla' => ['Quesadilla', 4500],
    'agua' => ['Agua de horchata', 2500],
];

$pedido = json_decode(file_get_contents('php://input'), true);
$productos = is_array($pedido['productos'] ?? null) ? $pedido['productos'] : [];
$operaciones = [
    ['tipo' => 'texto', 'valor' => 'TAQUERÍA EL EJEMPLO', 'alineacion' => 'centro', 'negrita' => true, 'tamano' => [2, 2]],
    ['tipo' => 'texto', 'valor' => 'Pedido en línea', 'alineacion' => 'centro'],
    ['tipo' => 'separador'],
];
$total = 0;
foreach ($productos as $p) {
    $clave = is_array($p) ? ($p['clave'] ?? '') : '';
    $cantidad = is_array($p) ? (int) ($p['cantidad'] ?? 0) : 0;
    if (!isset(CATALOGO[$clave]) || $cantidad < 1 || $cantidad > 50) {
        responderError(400, 'Producto o cantidad no válidos.');
    }
    [$nombre, $centavos] = CATALOGO[$clave];
    $total += $centavos * $cantidad;
    $operaciones[] = ['tipo' => 'par', 'izquierda' => "{$cantidad} x {$nombre}", 'derecha' => pesos($centavos * $cantidad)];
}
if ($total === 0) {
    responderError(400, 'El pedido está vacío.');
}
$folio = 'WEB-' . strtoupper(bin2hex(random_bytes(3)));
array_push(
    $operaciones,
    ['tipo' => 'separador'],
    ['tipo' => 'par', 'izquierda' => 'TOTAL', 'derecha' => pesos($total), 'negrita' => true],
    ['tipo' => 'texto', 'valor' => "Folio {$folio}", 'alineacion' => 'centro'],
    ['tipo' => 'cortar'],
);

function pesos(int $centavos): string
{
    return '$' . number_format($centavos / 100, 2);
}

$documento = ['version' => '1', 'impresora' => $config['impresora'], 'referencia' => $folio, 'operaciones' => $operaciones];
$llave = @file_get_contents($config['llave_privada']);
if ($llave === false) {
    responderError(500, 'No se encontró la llave privada; revisa config.php.');
}
// El id lo genera el firmador. Vigencia de 90 s: el navegador debe enviarlo enseguida.
echo json_encode(
    PrismaFirmador::firmar($documento, $config['cliente_id'], $llave, 90),
    JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES,
);
