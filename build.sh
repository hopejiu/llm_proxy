rm ./server*
GOOS=windows GOARCH=amd64 go build -o server.exe  ./cmd/server