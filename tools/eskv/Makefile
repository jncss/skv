.PHONY: build run clean install test help

# Variables
BINARY_NAME=eskv
GO=go
GOFLAGS=-v

# Ajuda per defecte
help:
	@echo "Makefile per eskv - Explorador web SKV"
	@echo ""
	@echo "Targets disponibles:"
	@echo "  make build    - Compila l'aplicació"
	@echo "  make run      - Executa l'aplicació"
	@echo "  make clean    - Esborra els binaris i fitxers temporals"
	@echo "  make install  - Instal·la les dependències"
	@echo "  make test     - Executa els tests"
	@echo "  make dev      - Compila i executa en mode desenvolupament"
	@echo ""

# Compilar l'aplicació
build:
	@echo "🔨 Compilant $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "✅ Compilació completada: ./$(BINARY_NAME)"

# Executar l'aplicació
run:
	@echo "🚀 Executant $(BINARY_NAME)..."
	$(GO) run .

# Desenvolupament: compilar i executar
dev: build
	@echo "🚀 Executant $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Esborrar binaris i fitxers temporals
clean:
	@echo "🧹 Netejant fitxers..."
	rm -f $(BINARY_NAME)
	rm -f *.skv *.wal
	@echo "✅ Neteja completada"

# Instal·lar dependències
install:
	@echo "📦 Instal·lant dependències..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependències instal·lades"

# Executar tests
test:
	@echo "🧪 Executant tests..."
	$(GO) test -v ./...

# Compilar per diferents plataformes
build-all:
	@echo "🔨 Compilant per a totes les plataformes..."
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "✅ Compilació multi-plataforma completada"
