package main

import (
	"fmt"
	"flag"
	"net"
	"math/rand"
	"time"
	"strconv"
	"github.com/dedis/protobuf"
	"encoding/hex"
	"strings"
)

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

type SearchRequest struct {
	Origin string
	Budget uint64
	Keywords []string
}

type SearchReply struct {
	Origin string
	Destination string
	HopLimit uint32
	Results []*SearchResult
}

type SearchResult struct {
	FileName string
	MetafileHash []byte
	ChunkMap []uint64
	ChunkCount uint64
}

type TxPublish struct {
	File File
	HopLimit uint
}

type BlockPublish struct {
	Block Block
	HopLimit uint32
}

type File struct {
	Name   string
	Size   int64
	MetafileHash []byte
}

type Block struct {
	PrevHash [32]byte
	Nonce [32]byte
	Transactions []TxPublish
}

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
 	Status *StatusPacket
 	Private *PrivateMessage
 	DataRequest *DataRequest
	DataReply *DataReply
	SearchRequest *SearchRequest
	SearchReply *SearchReply
	TxPublish *TxPublish
	BlockPublish *BlockPublish
}

func main() {
	var uiport = flag.String("UIPort", "8080", "port for client")
	var message = flag.String("msg", "", "message to be sent")
	var dest = flag.String("dest", "", "destination for private message of file request")
	var file = flag.String("file", "", "file to be indexed or requested")
	var request = flag.String("request", "", "hash of metafile")
	var keywords = flag.String("keywords", "", "file to be indexed or requested")
	var budget = flag.Uint64("budget", 0, "hash of metafile")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*uiport)
	if err != nil {
        fmt.Println(err)
        return
    }

    localAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(rand.Intn(10000) + 10000))
	if err != nil {
        fmt.Println(err)
        return
    }

	udpConn, err := net.DialUDP("udp4", localAddr, udpAddr)
	if err != nil {
        fmt.Println(err)
        return
    }
    defer udpConn.Close()

    packet := GossipPacket{}
    if *dest == "" {
    	if *file != "" {
    		mess := DataRequest{Origin: *file}
    		if  *request != "" {
    			if len(*request) != 64 {
				fmt.Println("Wrong length of hash")
				return
				}
				decoded_hash, err := hex.DecodeString(*request)
				if err != nil {
					fmt.Println(err)
				    return
				}
				mess.HashValue = decoded_hash
    		}
	    	packet.DataRequest = &mess
    	} else if *keywords != "" {
			kw := strings.Split(*keywords, ",")
			mess := SearchRequest{Budget: *budget, Keywords: kw}
	    	packet.SearchRequest = &mess
		} else {
	    	mess := SimpleMessage{Contents: *message}
	    	packet.Simple = &mess
	    }
	} else {
		if *file != "" && *request != "" {
			if len(*request) != 64 {
				fmt.Println("Wrong length of hash")
				return
			}
			decoded_hash, err := hex.DecodeString(*request)
			if err != nil {
				fmt.Println(err)
			    return
			}
			mess := DataRequest{Origin: *file, Destination: *dest, HashValue: decoded_hash}
	    	packet.DataRequest = &mess
		} else {
			mess := PrivateMessage{ID: 0, Text: *message, Destination: *dest, HopLimit: 10}
		    packet.Private = &mess
		}
	}
    packetBytes, err := protobuf.Encode(&packet)
	if err != nil {
        fmt.Println(err)
        return
    }
    udpConn.Write(packetBytes)
}
