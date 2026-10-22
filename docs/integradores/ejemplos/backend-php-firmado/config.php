<?php

// Configuración del ejemplo. En producción, toma estos valores de variables de entorno o de tu
// gestor de secretos; la llave privada nunca debe estar dentro de la carpeta pública del sitio.
return [
    // Dirección del puente PRISMA tal como la ve el navegador del negocio.
    'puente' => getenv('PRISMA_URL') ?: 'https://192.168.1.10:8443',
    // Alias de la impresora en el puente.
    'impresora' => getenv('PRISMA_IMPRESORA') ?: 'caja',
    // Id de la aplicación de nivel 3 (lo muestra el panel del puente, sección Aplicaciones).
    'cliente_id' => getenv('PRISMA_CLIENTE_ID') ?: 'cli_xxxxxxxxxxxxxxxx',
    // Llave privada Ed25519 (PEM PKCS #8). La pública correspondiente se registra en el panel.
    'llave_privada' => getenv('PRISMA_LLAVE_PRIVADA') ?: __DIR__ . '/llaves/privada.pem',
];
