rm -f ./server* ./llm-proxy*
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o server.exe ./cmd/server