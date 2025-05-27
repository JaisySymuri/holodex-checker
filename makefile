BINARY=holodex-checker-windows.exe

compile:
	go build -o $(BINARY) ./...
