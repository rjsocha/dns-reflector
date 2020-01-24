# dns-reflector
Simple DNS server which returns client IP address for A quieries.

Build
```
make
```

Install (tested on Ubuntu only with systemd)

```
sudo dpkg -i dist/dns-reflector_1.0_amd64.deb
# Replace 127.0.0.1 with your "PUBLIC" ip address
sudo systemctl enable dns-reflector@127.0.0.1
sudo systemctl start dns-reflector@127.0.0.1
```

Test
```
dig @127.0.0.1 . +short
dig @127.0.0.1 . txt +short 
dig @127.0.0.1 . txt +short +tcp

```
