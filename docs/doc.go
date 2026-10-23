// Package docs embebe el manual que muestra el panel y prueba que la documentación siga fiel al
// código. Las rutas de docs/referencia/openapi.yaml deben ser las de la API, cada código de error
// del puente y del SDK debe estar en docs/referencia/codigos-de-error.md, y los enlaces y anclas de
// todos los Markdown del repositorio deben existir.
package docs

import "embed"

// Manual son los documentos que el panel muestra en Ayuda, sin Internet: manual.json (los grupos y
// el orden) y los Markdown e imágenes de las carpetas que cita. docs/ es la única fuente: no hay
// copias ni pasos de generación. TestIndiceDelManual exige que cada archivo del índice exista.
//
//go:embed manual.json administracion/*.md administracion/img/*/*.webp instalacion/*.md operacion/*.md
//go:embed seguridad/*.md integradores/*.md referencia/*.md diagramas/*.md
var Manual embed.FS
