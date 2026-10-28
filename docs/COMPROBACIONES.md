# Catálogo de comprobaciones

## Implementadas

| ID | Severidad | Qué detecta | Cómo corregirlo |
|---|---|---|---|
| `connect` | critical | No se pudo abrir la conexión TLS o completar el handshake. | Verifica conectividad, puerto y estado del servicio. |
| `expiry` | warning / critical | Días restantes por debajo de los umbrales configurados; un valor negativo indica caducidad. | Renueva y despliega el certificado antes de su vencimiento. |
| `chain` | critical | La cadena presentada no valida con las raíces de confianza del sistema. | Instala y sirve la cadena correcta, incluidos los intermedios. |
| `hostname` | critical | El nombre solicitado no valida contra el certificado. | Emite un certificado con el SAN correcto y despliega ese certificado. |

En M1, `chain` y `hostname` aparecen como un campo de validación combinado. El catálogo individual se implementará en M2.

## Planificadas

Las comprobaciones de lifetime, cadena incompleta, fortaleza de clave, algoritmo de firma, autofirma, SCT, protocolos, cifrados, despliegue y Certificate Transparency se incorporarán en sus hitos correspondientes.