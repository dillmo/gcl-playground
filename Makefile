GO=tinygo build
GOFLAGS=-panic=trap
STRIP=wasm-strip
SRCFILE=src/main.go

public/main.wasm: $(SRCFILE)
		$(GO) $(GOFLAGS) -o $@ $(SRCFILE)
		$(STRIP) $@ 
