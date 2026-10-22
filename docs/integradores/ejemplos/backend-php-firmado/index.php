<?php
// Página del pedido. Solo expone la dirección del puente y el alias: nada secreto.
$config = require __DIR__ . '/config.php';
$h = fn (string $s): string => htmlspecialchars($s, ENT_QUOTES, 'UTF-8');
?>
<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PRISMA · pedido firmado (nivel 3)</title>
  <style>
    body { font: 16px/1.5 system-ui, sans-serif; max-width: 30rem; margin: 2rem auto; padding: 0 1rem; }
    label { display: flex; justify-content: space-between; margin: .5rem 0; }
    input { width: 4rem; }
    #resultado.error { color: #b00020; }
  </style>
</head>
<body>
  <h1>Pedido</h1>
  <form id="pedido">
    <label>Taco al pastor ($30.00) <input type="number" name="taco" min="0" max="50" value="2"></label>
    <label>Quesadilla ($45.00) <input type="number" name="quesadilla" min="0" max="50" value="1"></label>
    <label>Agua de horchata ($25.00) <input type="number" name="agua" min="0" max="50" value="0"></label>
    <button type="submit">Imprimir pedido</button>
  </form>
  <p id="resultado" role="status"></p>

  <script src="https://cdn.jsdelivr.net/gh/bayalad7/prisma@v0.1.0/sdk/js/dist/prisma.umd.js"></script>
  <script>
    // Nivel 3: sin llave en el navegador. El puente verifica la firma que hizo el backend.
    const prisma = new Prisma({ url: <?= json_encode($config['puente']) ?> });
    const resultado = document.getElementById('resultado');

    document.getElementById('pedido').addEventListener('submit', async (ev) => {
      ev.preventDefault();
      const productos = [...new FormData(ev.target)]
        .map(([clave, cantidad]) => ({ clave, cantidad: Number(cantidad) }))
        .filter((p) => p.cantidad > 0);
      resultado.className = '';
      resultado.textContent = 'Firmando…';
      try {
        // 1. El backend arma y firma el ticket.
        const respuesta = await fetch('firmar.php', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ productos }) });
        const firmado = await respuesta.json();
        if (!respuesta.ok) throw new Error(firmado.mensaje);
        // 2. El navegador lo entrega al puente sin modificarlo.
        resultado.textContent = 'Imprimiendo…';
        const r = await prisma.imprimirDocumento(firmado);
        resultado.textContent = `Impreso en «${r.impresora}» (${r.trabajo_id}, folio ${firmado.referencia}).`;
      } catch (e) {
        resultado.className = 'error';
        resultado.textContent = e instanceof Prisma.PrismaError ? `${e.mensaje} ${e.sugerencia}` : e.message;
      }
    });
  </script>
</body>
</html>
