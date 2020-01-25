all:	package/usr/sbin/dns-reflector
package/usr/sbin/dns-reflector: src/dns-reflector.go
	go build -o package/usr/sbin/dns-reflector -ldflags="-s -w" src/dns-reflector.go
	upx --brute package/usr/sbin/dns-reflector
	dpkg-deb --build package/ dist
docker-static:
	GO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o dns-reflector-static src/dns-reflector.go
	strip -x dns-reflector-static
	upx dns-reflector-static
	docker build -t dns-reflector-static -f Dockerfile.static .
