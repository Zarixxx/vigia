# Decisiones

- El módulo se llama `vigia` para mantener los imports internos cortos; el binario de usuario también se llama `vigia`.
- M1 mantiene tanto `vigia check ...` como el uso directo del apéndice para facilitar la transición.
- La salida CSV y Prometheus queda para los hitos que incorporan esos formatos; M1 admite table y JSON.
- No se usan dependencias externas. Los tests generan certificados localmente y no requieren acceso a Internet.