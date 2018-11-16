package main

import (
	"fmt"
	"net"
	"github.com/dedis/protobuf"
)

func sendMessage(packetToSend *GossipPacket, addr string, socket *net.UDPConn) {
	if addr == "" {
		return
	}
	packetBytes, err := protobuf.Encode(packetToSend)
	if err != nil {
        fmt.Println(err)
        return
    }
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
        fmt.Println(err)
        return
    }
	socket.WriteToUDP(packetBytes, udpAddr)
}

func listen(socket *net.UDPConn) (*GossipPacket, *net.UDPAddr, error) {
	buf := make([]byte, 16*1024)
	n,addr,err := socket.ReadFromUDP(buf)
	if err != nil {
        fmt.Println(err)
	}
	dec := &GossipPacket{}
	err = protobuf.Decode(buf[0:n], dec)
	if err != nil {
        fmt.Println(err)
	}
	return dec,addr,err
}

func checkAndAppend(list *[]string, addr string) {
	flag := false
	for _, v := range *list {
		if v == addr {
			flag = true
		}
	}
	if flag == false {
		*list = append(*list, addr)
	}
}
