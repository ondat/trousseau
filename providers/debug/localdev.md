# Local development

This document describes how to develop Trousseau Debug provider on your local machine.

Please follow base documentation at [localdev.md](../localdev.md)

## Run Trousseau components

Use command line or our favorite IDE to start Trousseau components on your machine:

```bash
mkdir bin/debug
(cd proxy ; go mod tidy && go run main.go --listen-addr unix://../bin/proxy.socket --trousseau-addr ../bin/trousseau.socket)
(cd providers/debug ; go mod tidy && go run main.go --listen-addr unix://../../bin/debug/debug.socket)
(cd trousseau ; go mod tidy && go run main.go --enabled-providers debug --socket-location ../bin --listen-addr unix://../bin/trousseau.socket --zap-encoder=console --v=5)
```
