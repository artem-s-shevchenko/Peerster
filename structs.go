package main

import (
	"sync"
	"net"
	"fmt"
	"time"
)

type IPMessage struct{
	IP string
}

type MessageMessage struct{
	Message string
}

type Update struct {
	Newmessages []string
	Peerlist []string
	Myid string
}

type SimpleMessage struct {
	OriginalName string
	RelayPeerAddr string
	Contents string
}

type RumorMessage struct {
	Origin string
 	ID uint32
	Text string
}

type PeerStatus struct {
	Identifier string
	NextID uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

type PrivateMessage struct {
	Origin string
 	ID uint32
	Text string
	Destination string
  	HopLimit uint32
}

type DataRequest struct{
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
}

type DataReply struct{
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
	Data []byte
}

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
 	Status *StatusPacket
 	Private *PrivateMessage
 	DataRequest *DataRequest
	DataReply *DataReply
}

type SafePeersList struct {
	PeersList []string
	mux sync.Mutex
}

type SafeMessageHistory struct {
	History map[string] []string
	mux sync.Mutex
}

type SafePrivateMessageHistory struct {
	History map[string] []string
	mux sync.Mutex
}

type SafeUpdateMessageList struct {
	List []string
	mux sync.Mutex
}

type Timeout struct {
	Timer *time.Timer
	Origin string
	ID uint32
	Text string
}

type SafeTimeoutMap struct {
	TimeoutMap map[string] []Timeout
	mux sync.Mutex
}

type RoutingTable struct {
	RouteTable map[string]string
	IdTable map[string]uint32
	mux sync.Mutex
}

type SafeFileIndex struct {
	FileIndex map[[32]byte] []byte
	mux sync.Mutex
}

type Gossiper struct {
	ClientConn *net.UDPConn
	PeerConn *net.UDPConn
	Nodename string
	GossipAddress string
	SafePeersList *SafePeersList
	SafeMessageHistory *SafeMessageHistory
	SafeTimeoutMap *SafeTimeoutMap
	SafeUpdateMessageList *SafeUpdateMessageList
	id uint32
	RoutingTable *RoutingTable
	SafePrivateMessageHistory *SafePrivateMessageHistory
	SafeFileIndex *SafeFileIndex
}

func NewGossiper(clientAddress, peerAddress, name string, peersList []string) *Gossiper {
	udpClientAddr, err := net.ResolveUDPAddr("udp4", clientAddress)
	if err != nil {
            fmt.Println(err)
    }
	udpClientConn, err := net.ListenUDP("udp4", udpClientAddr)
	if err != nil {
            fmt.Println(err)
    }
    udpPeerAddr, err := net.ResolveUDPAddr("udp4", peerAddress)
	if err != nil {
            fmt.Println(err)
    }
	udpPeerConn, err := net.ListenUDP("udp4", udpPeerAddr)
	if err != nil {
            fmt.Println(err)
    }
	return &Gossiper{
		udpClientConn,
		udpPeerConn,
		name,
		peerAddress,
		&SafePeersList{PeersList: peersList},
		&SafeMessageHistory{History: map[string] []string{}},
		&SafeTimeoutMap{TimeoutMap: map[string] []Timeout{}},
		&SafeUpdateMessageList{List: []string{}},
		0,
		&RoutingTable{RouteTable:map[string]string{}, IdTable:map[string]uint32{}},
		&SafePrivateMessageHistory{History: map[string] []string{}},
		&SafeFileIndex{FileIndex: map[[32]byte] []byte{}}}
}
