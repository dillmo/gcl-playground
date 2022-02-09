GO=tinygo build
GOFLAGS=-panic=trap
STRIP=wasm-strip
SRCFILES=src/main.go src/lex/lex.go src/parse/parse.go

public/main.wasm: $(SRCFILES)
		$(GO) $(GOFLAGS) -o $@ src/main.go
		$(STRIP) $@ 
