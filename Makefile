all:	package/usr/sbin/dns-reflector
package/usr/sbin/dns-reflector: src/dns-reflector.go
	go build -o package/usr/sbin/dns-reflector -ldflags="-s -w" src/dns-reflector.go
	upx --brute package/usr/sbin/dns-reflector
	dpkg-deb --build package/ dist
