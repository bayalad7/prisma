<?php

declare(strict_types=1);

/**
 * Firmador de referencia de PRISMA (nivel 3) para PHP 8.2+ con la extensión sodium.
 *
 *     require 'PrismaFirmador.php';
 *     $firmado = PrismaFirmador::firmar($documento, 'cli_…', file_get_contents('privada.pem'));
 *
 * Firma con Ed25519 la forma canónica JCS (RFC 8785) del documento sin «firma», como verifica el
 * puente (docs/contrato-documento-v1.md §7). Sin dependencias de Composer.
 */
final class PrismaFirmador
{
    public const VIGENCIA_MAXIMA = 120;

    /** Prefijo DER de una llave privada Ed25519 en PKCS #8 (RFC 8410); le siguen los 32 bytes de la semilla. */
    private const PREFIJO_PKCS8 = "\x30\x2e\x02\x01\x00\x30\x05\x06\x03\x2b\x65\x70\x04\x22\x04\x20";

    /**
     * Completa y firma un documento v1: agrega cliente_id, emitido, expira y firma, y un id UUID v4
     * si falta. Ignora una firma anterior.
     *
     * @param array<string, mixed> $documento documento v1 decodificado con json_decode(..., true)
     * @return array<string, mixed>
     */
    public static function firmar(
        array $documento,
        string $clienteId,
        string $llavePrivadaPem,
        int $vigencia = 90,
        ?DateTimeInterface $ahora = null,
    ): array {
        if ($vigencia < 1 || $vigencia > self::VIGENCIA_MAXIMA) {
            throw new InvalidArgumentException('La vigencia debe ser de 1 a ' . self::VIGENCIA_MAXIMA . ' segundos.');
        }
        if (preg_match('/^cli_[A-Za-z0-9]{8,40}$/', $clienteId) !== 1) {
            throw new InvalidArgumentException('El cliente debe tener la forma cli_… (el id que muestra el panel).');
        }
        $emitido = DateTimeImmutable::createFromInterface($ahora ?? new DateTimeImmutable())
            ->setTimezone(new DateTimeZone('UTC'));
        $emitido = $emitido->setTime((int) $emitido->format('H'), (int) $emitido->format('i'), (int) $emitido->format('s'));

        $doc = $documento;
        unset($doc['firma']);
        $doc['id'] ??= self::uuid4();
        $doc['cliente_id'] = $clienteId;
        $doc['emitido'] = $emitido->format('Y-m-d\TH:i:s\Z');
        $doc['expira'] = $emitido->modify("+{$vigencia} seconds")->format('Y-m-d\TH:i:s\Z');

        $firma = sodium_crypto_sign_detached(self::canonicalizar($doc), self::llaveSecreta($llavePrivadaPem));
        $doc['firma'] = base64_encode($firma);
        return $doc;
    }

    /**
     * Forma canónica JCS (RFC 8785). Acepta arreglos asociativos y objetos (stdClass, por ejemplo de
     * json_decode sin true). Un arreglo vacío se serializa como [] y un stdClass vacío como {}.
     */
    public static function canonicalizar(mixed $valor): string
    {
        return match (true) {
            $valor === null => 'null',
            is_bool($valor) => $valor ? 'true' : 'false',
            is_int($valor) => (string) $valor,
            is_float($valor) => self::numero($valor),
            is_string($valor) => self::cadena($valor),
            is_array($valor) && array_is_list($valor) => '[' . implode(',', array_map(self::canonicalizar(...), $valor)) . ']',
            is_array($valor), $valor instanceof stdClass => self::objeto((array) $valor),
            default => throw new InvalidArgumentException('JCS no admite valores de tipo ' . get_debug_type($valor) . '.'),
        };
    }

    /** @param array<int|string, mixed> $miembros */
    private static function objeto(array $miembros): string
    {
        $claves = array_map('strval', array_keys($miembros));
        // JCS ordena por unidades de código UTF-16: comparar los bytes en UTF-16BE da ese orden.
        usort($claves, static fn (string $a, string $b): int => strcmp(
            mb_convert_encoding($a, 'UTF-16BE', 'UTF-8'),
            mb_convert_encoding($b, 'UTF-16BE', 'UTF-8'),
        ));
        $partes = [];
        foreach ($claves as $clave) {
            $partes[] = self::cadena($clave) . ':' . self::canonicalizar($miembros[$clave]);
        }
        return '{' . implode(',', $partes) . '}';
    }

    /** Escape mínimo de JSON; el resto del texto va en UTF-8 directo. */
    private static function cadena(string $s): string
    {
        if (!mb_check_encoding($s, 'UTF-8')) {
            throw new InvalidArgumentException('Las cadenas deben estar en UTF-8.');
        }
        $cortos = ['"' => '\\"', '\\' => '\\\\', "\x08" => '\\b', "\t" => '\\t', "\n" => '\\n', "\x0c" => '\\f', "\r" => '\\r'];
        $salida = preg_replace_callback(
            '/[\x00-\x1f"\\\\]/',
            static fn (array $m): string => $cortos[$m[0]] ?? sprintf('\\u%04x', ord($m[0])),
            $s,
        );
        return '"' . $salida . '"';
    }

    /** Los documentos v1 solo llevan enteros; un float entero (2.0) se escribe como 2. */
    private static function numero(float $n): string
    {
        if (!is_finite($n) || floor($n) !== $n || abs($n) > 9007199254740991) {
            throw new InvalidArgumentException("El documento no puede llevar el número {$n}: el contrato v1 solo usa enteros.");
        }
        return (string) (int) $n;
    }

    /** Llave secreta de sodium (semilla + pública) a partir del PEM PKCS #8. */
    private static function llaveSecreta(string $pem): string
    {
        if (preg_match('/-----BEGIN PRIVATE KEY-----(.+?)-----END PRIVATE KEY-----/s', $pem, $m) !== 1) {
            throw new InvalidArgumentException('La llave privada debe ser un PEM «BEGIN PRIVATE KEY» (PKCS #8).');
        }
        $der = base64_decode(preg_replace('/\s+/', '', $m[1]), true);
        if ($der === false || strlen($der) !== 48 || !str_starts_with($der, self::PREFIJO_PKCS8)) {
            throw new InvalidArgumentException('La llave privada no es Ed25519.');
        }
        return sodium_crypto_sign_secretkey(sodium_crypto_sign_seed_keypair(substr($der, 16)));
    }

    private static function uuid4(): string
    {
        $b = random_bytes(16);
        $b[6] = chr((ord($b[6]) & 0x0f) | 0x40);
        $b[8] = chr((ord($b[8]) & 0x3f) | 0x80);
        return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($b), 4));
    }
}
