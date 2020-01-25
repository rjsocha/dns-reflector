// Modified by Robert Socha 
// Ripped from: https://github.com/miekg/exdns
//
// Changes:
// added -listen directive
// any zone supported
// ex:
//    ./dns-reflector -listen <IP>:53,127.0.0.1:53
//
// Copyright 2011 Miek Gieben. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Reflect is a small name server which sends back the IP address of its client, the
// recursive resolver.
// When queried for type A (resp. AAAA), it sends back the IPv4 (resp. v6) address.
// In the additional section the port number and transport are shown.
//
// Basic use pattern:
//
//	dig @localhost -p 8053 whoami.miek.nl A
//
//	;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 2157
//	;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1
//	;; QUESTION SECTION:
//	;whoami.miek.nl.			IN	A
//
//	;; ANSWER SECTION:
//	whoami.miek.nl.		0	IN	A	127.0.0.1
//
//	;; ADDITIONAL SECTION:
//	whoami.miek.nl.		0	IN	TXT	"Port: 56195 (udp)"
//
// Similar services: whoami.ultradns.net, whoami.akamai.net. Also (but it
// is not their normal goal): rs.dns-oarc.net, porttest.dns-oarc.net,
// amiopen.openresolvers.org.
//
// Original version is from: Stephane Bortzmeyer <stephane+grong@bortzmeyer.org>.
//
// Adapted to Go (i.e. completely rewritten) by Miek Gieben <miek@miek.nl>.
package main


import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"os/signal"
	"runtime"
	"syscall"
	"io/ioutil"
	"github.com/miekg/dns"
)

var (
	compress    = flag.Bool("compress", false, "compress replies")
	soreuseport = flag.Int("soreuseport", 0, "use SO_REUSE_PORT")
	cpu         = flag.Int("cpu", 0, "number of cpu to use")
	listen	    = flag.String("listen","[::]:53","Listen on specific address(es) -> listen1,listen2,listenN")
	version     = flag.Bool("version",false,"show app version")
	client	    = flag.String("client","","say hello from cmd line")
	socket	    = flag.String("socket","/dns.sock","managment socket")
)

const  Version = "1.0"

func handleReflect(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4  bool
		rr  dns.RR
		str string
		a   net.IP
	)
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = *compress
	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		str = ip.String() + "@udp"
		a = ip.IP
		v4 = a.To4() != nil
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		str = ip.String() + "@tcp"
		a = ip.IP
		v4 = a.To4() != nil
	}
	if v4 {
		rr = &dns.A{
			Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   a.To4(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr:  dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
			AAAA: a,
		}
	}

	t := &dns.TXT{
		Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: []string{str},
	}

	switch r.Question[0].Qtype {
	case dns.TypeTXT:
		m.Answer = append(m.Answer, t)
	//	m.Extra = append(m.Extra, rr)
	case dns.TypeAAAA, dns.TypeA,dns.TypeANY:
		m.Answer = append(m.Answer, rr)
	//	m.Extra = append(m.Extra, t)
	}
	w.WriteMsg(m)
	fmt.Println("Served DNS query for ",m.Question[0].Name)
}

func serve(net,listen string, soreuseport bool) {
	server := &dns.Server{Addr: listen, Net: net, TsigSecret: nil, ReusePort: soreuseport}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the %s server on %s:\n\t%s\n",net,listen,err.Error())
		os.Exit(1)
	}
}

func cmdServer(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		if(string(data) == "ls") {
			files, err := ioutil.ReadDir("/")
			if err == nil {
				for _, f := range files {
			                fmt.Println(f.Name())
				}
			}
		} else {
			fmt.Println("Server got:", string(data))
		}
		//_, err = c.Write(data)
		//if err != nil {
		//	log.Fatal("Writing client error: ", err)
		//}
	}
}

func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	if(len(*client)>0) {
		c, err := net.Dial("unix", *socket)
		if err != nil {
			fmt.Println("Dial error", err)
			os.Exit(1)
		}
		defer c.Close()
		fmt.Printf("Client Mode enabled.\n")
		fmt.Printf("Sending to the server: %s\n",*client);
		c.Write([]byte(*client))
		os.Exit(1)
	}
	if(*version) {
		fmt.Printf("%s\n",Version)
		os.Exit(1)
	}
	if *cpu != 0 {
		runtime.GOMAXPROCS(*cpu)
	}
	dns.HandleFunc(".", handleReflect)
	listeners := strings.Split(*listen,",")
	for idx := range listeners {
		addr := listeners[idx]
		host, port,err := net.SplitHostPort(addr)
		_ = host
		_ = port
		if(err == nil ) {
			if *soreuseport > 0 {
				for i := 0; i < *soreuseport; i++ {
					go serve("tcp", addr,true)
					go serve("udp", addr,true)
				}
			} else {
				go serve("tcp", addr,false)
				go serve("udp", addr,false)
			}

		} else {
			fmt.Println("Unable to parse listen address ->",err)
			os.Exit(1)
		}
	}
	ln,err := net.Listen("unix",*socket)
	if err != nil {
		fmt.Println("Listen ",*socket," error: ",err)
		os.Exit(1)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func(ln net.Listener, c chan os.Signal) {
		sig := <-c
		fmt.Printf("Caught signal %s: shutting down.\n", sig)
		os.Exit(0)
		ln.Close()
	}(ln, sig)

	for {
		fd, err := ln.Accept()
		if err != nil {
			fmt.Printf("Accept error: ", err)
		}
		cmdServer(fd)
	}

	s := <-sig
	fmt.Printf("Bye... (%s)\n", s)
}
