"""Firmador de referencia de PRISMA (nivel 3) para Python 3.9+ con la biblioteca cryptography.

    from prisma_firmador import firmar
    firmado = firmar(documento, "cli_…", open("privada.pem", "rb").read())

Firma con Ed25519 la forma canónica JCS (RFC 8785) del documento sin «firma», como verifica el
puente (docs/contrato-documento-v1.md §7). Única dependencia: pip install cryptography.
"""

from __future__ import annotations

import base64
import json
import re
import uuid
from datetime import datetime, timedelta, timezone
from typing import Any, Optional, Union

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import load_pem_private_key

__all__ = ["canonicalizar", "firmar", "VIGENCIA_MAXIMA"]

VIGENCIA_MAXIMA = 120
_PATRON_CLIENTE = re.compile(r"^cli_[A-Za-z0-9]{8,40}$")


def canonicalizar(valor: Any) -> str:
    """Forma canónica JCS: claves ordenadas por unidades UTF-16, sin espacios y escape mínimo."""
    if valor is None or isinstance(valor, bool):
        return json.dumps(valor)
    if isinstance(valor, int):
        return str(valor)
    if isinstance(valor, float):
        # Los documentos v1 solo llevan enteros; un float entero (2.0) se escribe como 2.
        if not valor.is_integer() or abs(valor) > 9007199254740991:
            raise ValueError(f"El documento no puede llevar el número {valor}: el contrato v1 solo usa enteros.")
        return str(int(valor))
    if isinstance(valor, str):
        # Con ensure_ascii=False, json.dumps hace el escape mínimo que pide JCS.
        return json.dumps(valor, ensure_ascii=False)
    if isinstance(valor, (list, tuple)):
        return "[" + ",".join(canonicalizar(v) for v in valor) + "]"
    if isinstance(valor, dict):
        claves = sorted(valor, key=lambda k: k.encode("utf-16-be"))
        return "{" + ",".join(f"{json.dumps(k, ensure_ascii=False)}:{canonicalizar(valor[k])}" for k in claves) + "}"
    raise TypeError(f"JCS no admite valores de tipo {type(valor).__name__}")


def _fecha(d: datetime) -> str:
    return d.strftime("%Y-%m-%dT%H:%M:%SZ")


def firmar(
    documento: dict,
    cliente_id: str,
    llave_privada: Union[bytes, str, Ed25519PrivateKey],
    vigencia: int = 90,
    ahora: Optional[datetime] = None,
) -> dict:
    """Completa y firma un documento v1: agrega cliente_id, emitido, expira y firma, y un id UUID v4
    si falta. Devuelve un diccionario nuevo; ignora una firma anterior."""
    if not isinstance(vigencia, int) or not 1 <= vigencia <= VIGENCIA_MAXIMA:
        raise ValueError(f"La vigencia debe ser de 1 a {VIGENCIA_MAXIMA} segundos.")
    if not _PATRON_CLIENTE.match(cliente_id or ""):
        raise ValueError("El cliente debe tener la forma cli_… (el id que muestra el panel).")
    if isinstance(llave_privada, Ed25519PrivateKey):
        llave = llave_privada
    else:
        pem = llave_privada.encode() if isinstance(llave_privada, str) else llave_privada
        llave = load_pem_private_key(pem, password=None)
        if not isinstance(llave, Ed25519PrivateKey):
            raise ValueError("La llave privada debe ser Ed25519.")

    emitido = (ahora or datetime.now(timezone.utc)).astimezone(timezone.utc).replace(microsecond=0)
    doc = {k: v for k, v in documento.items() if k != "firma"}
    doc.setdefault("id", str(uuid.uuid4()))
    doc["cliente_id"] = cliente_id
    doc["emitido"] = _fecha(emitido)
    doc["expira"] = _fecha(emitido + timedelta(seconds=vigencia))
    firma = llave.sign(canonicalizar(doc).encode("utf-8"))
    doc["firma"] = base64.b64encode(firma).decode("ascii")
    return doc
