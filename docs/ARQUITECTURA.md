# Arquitectura

## Alcance actual

El ejecutable `cmd/vigia` delega el procesamiento de argumentos en `internal/app`. El paquete `internal/probe` normaliza objetivos, ejecuta handshakes TLS con contexto y límite de concurrencia, y calcula los datos del certificado presentado. `internal/checks` contiene la clasificación de vencimiento; `internal/output` ordena y presenta resultados como tabla o JSON. El reloj está abstraído en `internal/clock`.

`InsecureSkipVerify` solo aparece en la sonda: permite observar certificados vencidos o inválidos. La cadena y el nombre se validan después de recibir el certificado. Una prueba AST restringe el uso de esa opción al método de sondeo.

## Evolución

Los hitos siguientes añadirán configuración, comprobaciones, inventario, alertas, daemon y panel. No forman parte de esta versión inicial.