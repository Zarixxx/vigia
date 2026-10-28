- [x] Verify that the copilot-instructions.md file in the .github directory is created.
- [x] Clarify Project Requirements: proyecto CLI de Vigía en Go 1.22, definido por el usuario.
- [x] Scaffold the Project: estructura M0/M1 creada en el workspace existente.
- [x] Customize the Project: núcleo de sonda TLS y CLI M1 implementados.
- [x] Install Required Extensions: no se requieren extensiones.
- [x] Compile the Project: compilación sin CGO y `vigia version` verificados con Go 1.22.12 local.
- [x] Create and Run Task: no se necesita una tarea de VS Code para este CLI.
- [ ] Launch the Project: no iniciado; requiere objetivos explícitos del usuario.
- [x] Ensure Documentation is Complete: README y documentación inicial creados.

## Guía del proyecto

- Mantén Go 1.22+, biblioteca estándar y `CGO_ENABLED=0`.
- Los identificadores van en inglés; los mensajes, comentarios de usuario y documentación, en español.
- No implementes hitos posteriores a M1 sin que el usuario continúe la tarea.
- Los tests de red usan servidores locales y nunca acceden a Internet.
- `InsecureSkipVerify` solo se permite en `internal/probe.Probe`, con validación manual posterior.