# Vigía

Vigía consulta el certificado X.509 que un servicio TLS presenta durante el handshake. En esta primera versión permite revisar objetivos explícitos, ver su vencimiento y obtener una validación básica de cadena y nombre.

## Requisitos

- Go 1.22 o superior para compilar desde el código fuente.
- No se requieren dependencias externas.

## Compilar

```sh
make build
```

El binario queda en `bin/vigia`. Para comprobar un servicio:

```sh
./bin/vigia check ejemplo.es
./bin/vigia check --format json --days 30 --crit 7 ejemplo.es:443
./bin/vigia check -f hosts.txt
```

También se conserva el uso directo `vigia [opciones] host[:puerto] ...`. Si no se indica puerto, se usa 443. Los archivos de objetivos aceptan un host por línea, líneas vacías y comentarios que comienzan con `#`.

## Códigos de salida

- `0`: todos los certificados están fuera del umbral de aviso.
- `1`: al menos un certificado está por debajo del umbral.
- `2`: no se pudo establecer una conexión TLS con algún objetivo.
- `3`: uso, opciones o archivo de objetivos incorrectos.

La validación de certificados se realiza después del handshake para poder informar sobre certificados caducados o autofirmados. Esta herramienta solo sondea los objetivos que se le pasan explícitamente; no descubre ni escanea redes.

## Estado

M0 y M1: andamiaje, reloj inyectable y comprobación TLS básica. Los hitos posteriores de la especificación aún no están implementados.

## Interfaz de demostración

El directorio `web/` contiene un prototipo autónomo del panel, con objetivos de muestra. No está conectado al sondeo, al inventario ni a una cuenta SaaS.

```sh
python3 -m http.server 4173 --directory web
```

Abre `http://localhost:4173` en el navegador. La interfaz permite buscar y filtrar la tabla de muestra, añadir objetivos locales y exportar CSV; no realiza conexiones de red.

Consulta [docs/ARQUITECTURA.md](docs/ARQUITECTURA.md), [docs/COMPROBACIONES.md](docs/COMPROBACIONES.md) y [DECISIONES.md](DECISIONES.md).