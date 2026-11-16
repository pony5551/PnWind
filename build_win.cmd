set GOARCH=amd64
set GOOS=windows
set GO111MODULE=on
set GOPROXY=https://goproxy.cn,direct
set GOFLAGS=-p=1

go build -o .\_bin\PnWind.exe -ldflags "-s -w" github.com/pony5551/PnWind
