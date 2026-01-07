package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	socks5Version = 0x05
	noAuth        = 0x00
	connectCmd    = 0x01
	ipv4Addr      = 0x01
	domainAddr    = 0x03
	ipv6Addr      = 0x04
)

func main() {
	listener, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("SOCKS5 proxy listening on :1080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	if err := handleHandshake(conn); err != nil {
		log.Printf("handshake error: %v", err)
		return
	}

	targetConn, err := handleRequest(conn)
	if err != nil {
		log.Printf("request error: %v", err)
		return
	}
	defer targetConn.Close()

	relay(conn, targetConn)
}

func handleHandshake(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != socks5Version {
		return errors.New("unsupported SOCKS version")
	}

	numMethods := int(buf[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	hasNoAuth := false
	for _, m := range methods {
		if m == noAuth {
			hasNoAuth = true
			break
		}
	}

	if !hasNoAuth {
		conn.Write([]byte{socks5Version, 0xFF})
		return errors.New("no acceptable auth method")
	}

	_, err := conn.Write([]byte{socks5Version, noAuth})
	return err
}

func handleRequest(conn net.Conn) (net.Conn, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	if buf[0] != socks5Version {
		return nil, errors.New("unsupported SOCKS version")
	}

	if buf[1] != connectCmd {
		sendReply(conn, 0x07, nil)
		return nil, errors.New("unsupported command")
	}

	var host string
	addrType := buf[3]

	switch addrType {
	case ipv4Addr:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
		host = net.IP(addr).String()

	case domainAddr:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return nil, err
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return nil, err
		}
		host = string(domain)

	case ipv6Addr:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, err
		}
		host = net.IP(addr).String()

	default:
		sendReply(conn, 0x08, nil)
		return nil, errors.New("unsupported address type")
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return nil, err
	}
	port := binary.BigEndian.Uint16(portBuf)

	target := fmt.Sprintf("%s:%d", host, port)
	targetConn, err := net.Dial("tcp", target)
	if err != nil {
		sendReply(conn, 0x05, nil)
		return nil, err
	}

	localAddr := targetConn.LocalAddr().(*net.TCPAddr)
	sendReply(conn, 0x00, localAddr)

	return targetConn, nil
}

func sendReply(conn net.Conn, status byte, addr *net.TCPAddr) {
	reply := make([]byte, 10)
	reply[0] = socks5Version
	reply[1] = status
	reply[2] = 0x00
	reply[3] = ipv4Addr

	if addr != nil {
		ip := addr.IP.To4()
		if ip != nil {
			copy(reply[4:8], ip)
		}
		binary.BigEndian.PutUint16(reply[8:10], uint16(addr.Port))
	}

	conn.Write(reply)
}

func relay(client, target net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(target, client)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(client, target)
		done <- struct{}{}
	}()

	<-done
}
