"""python -m unittest discover -s sdk/firmadores/python"""

import json
import unittest
from datetime import datetime, timezone, timedelta
from pathlib import Path

from prisma_firmador import canonicalizar, firmar

AQUI = Path(__file__).resolve().parent
VECTOR = AQUI.parent.parent.parent / "testdata" / "vector-firma"


def vector(nombre: str) -> str:
    return (VECTOR / nombre).read_text(encoding="utf-8")


LLAVE = vector("llave-privada.pem").encode()
AHORA = datetime(2026, 10, 14, 18, 30, tzinfo=timezone.utc)


class PruebasFirmador(unittest.TestCase):
    def test_forma_canonica_igual_a_la_del_puente(self):
        casos = json.loads((AQUI.parent / "casos-jcs.json").read_text(encoding="utf-8"))
        for caso in casos:
            with self.subTest(caso["nombre"]):
                self.assertEqual(canonicalizar(caso["entrada"]), caso["canonico"])

    def test_vector_fijo(self):
        firmado = firmar(json.loads(vector("documento.json")), "cli_7Hq2MZp4XwR9", LLAVE, ahora=AHORA)
        sin_firma = {k: v for k, v in firmado.items() if k != "firma"}
        self.assertEqual(canonicalizar(sin_firma), vector("canonico.txt"))
        self.assertEqual(firmado["firma"], vector("firma.txt").strip())
        self.assertEqual(firmado, json.loads(vector("firmado.json")))

    def test_volver_a_firmar_ignora_la_firma_anterior(self):
        firmado = firmar(json.loads(vector("firmado.json")), "cli_7Hq2MZp4XwR9", LLAVE, ahora=AHORA)
        self.assertEqual(firmado["firma"], vector("firma.txt").strip())

    def test_id_nuevo_vigencia_y_zona_horaria(self):
        doc = json.loads(vector("documento.json"))
        del doc["id"]
        local = datetime(2026, 10, 14, 13, 30, 0, 750000, tzinfo=timezone(timedelta(hours=-5)))
        firmado = firmar(doc, "cli_7Hq2MZp4XwR9", LLAVE.decode(), vigencia=120, ahora=local)
        self.assertRegex(firmado["id"], r"^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")
        self.assertEqual((firmado["emitido"], firmado["expira"]), ("2026-10-14T18:30:00Z", "2026-10-14T18:32:00Z"))
        for vigencia, cliente in ((121, "cli_7Hq2MZp4XwR9"), (90, "cliente")):
            with self.assertRaises(ValueError):
                firmar(doc, cliente, LLAVE, vigencia=vigencia)
        with self.assertRaises(ValueError):
            canonicalizar({"precio": 1.5})


if __name__ == "__main__":
    unittest.main()
