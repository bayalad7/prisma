<?php

// php sdk/firmadores/php/prueba.php — sale con código 1 si algo no coincide con el puente.
declare(strict_types=1);

require __DIR__ . '/PrismaFirmador.php';

$fallas = 0;
function comprobar(bool $ok, string $que): void
{
    global $fallas;
    echo ($ok ? 'ok    ' : 'FALLA ') . $que . PHP_EOL;
    $fallas += $ok ? 0 : 1;
}

$vector = fn (string $n): string => file_get_contents(__DIR__ . "/../../../testdata/vector-firma/{$n}");
$llave = $vector('llave-privada.pem');
$ahora = new DateTimeImmutable('2026-10-14T18:30:00Z');

// 1. Forma canónica igual a la del puente. stdClass conserva {} vacíos.
foreach (json_decode(file_get_contents(__DIR__ . '/../casos-jcs.json'), false, 512, JSON_THROW_ON_ERROR) as $caso) {
    comprobar(PrismaFirmador::canonicalizar($caso->entrada) === $caso->canonico, "JCS: {$caso->nombre}");
}

// 2. Vector fijo.
$firmado = PrismaFirmador::firmar(json_decode($vector('documento.json'), true), 'cli_7Hq2MZp4XwR9', $llave, 90, $ahora);
$sinFirma = $firmado;
unset($sinFirma['firma']);
comprobar(PrismaFirmador::canonicalizar($sinFirma) === $vector('canonico.txt'), 'vector: forma canónica igual a canonico.txt');
comprobar($firmado['firma'] === trim($vector('firma.txt')), 'vector: firma igual a firma.txt');
$esperado = json_decode($vector('firmado.json'), true);
ksort($esperado);
ksort($firmado);
comprobar($firmado == $esperado, 'vector: documento igual a firmado.json');

// 3. Volver a firmar ignora la firma anterior.
$otraVez = PrismaFirmador::firmar(json_decode($vector('firmado.json'), true), 'cli_7Hq2MZp4XwR9', $llave, 90, $ahora);
comprobar($otraVez['firma'] === trim($vector('firma.txt')), 'volver a firmar ignora la firma anterior');

// 4. Id nuevo, segundos truncados y límites.
$doc = json_decode($vector('documento.json'), true);
unset($doc['id']);
$nuevo = PrismaFirmador::firmar($doc, 'cli_7Hq2MZp4XwR9', $llave, 120, new DateTimeImmutable('2026-10-14T13:30:00.750-05:00'));
comprobar(preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/', $nuevo['id']) === 1, 'id UUID v4 cuando falta');
comprobar([$nuevo['emitido'], $nuevo['expira']] === ['2026-10-14T18:30:00Z', '2026-10-14T18:32:00Z'], 'emitido en UTC sin fracciones');
foreach ([[121, 'cli_7Hq2MZp4XwR9', $llave], [90, 'cliente', $llave], [90, 'cli_7Hq2MZp4XwR9', 'no es PEM']] as [$v, $c, $k]) {
    try {
        PrismaFirmador::firmar($doc, $c, $k, $v);
        comprobar(false, "rechaza vigencia={$v} cliente={$c}");
    } catch (InvalidArgumentException) {
        comprobar(true, "rechaza vigencia={$v} cliente={$c} llave inválida=" . ($k === $llave ? 'no' : 'sí'));
    }
}

echo $fallas === 0 ? "\nTodo coincide con el puente.\n" : "\n{$fallas} falla(s).\n";
exit($fallas === 0 ? 0 : 1);
