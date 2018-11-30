package main

import (
	"fmt"
	"os"
	"io"
	"crypto/sha256"
	"encoding/hex"
	"time"
	"path/filepath"
	"math/rand"
)

var file_timeout int = 5

func index_file(gossiper *Gossiper, path string) {
	ex, err := os.Executable()
    if err != nil {
        fmt.Println(err)
    }
    exPath := filepath.Dir(ex)
    abs_path := filepath.Join(exPath, "_SharedFiles", path)
	fmt.Println("INDEXING", path)
	BufferSize := 8192
	metafile := []byte{}
	file, err := os.Open(abs_path)
	if err != nil {
	    fmt.Println(err)
	    return
	}
	defer file.Close()
	for {
		buffer := make([]byte, BufferSize)
	    bytesread, err := file.Read(buffer)
	    if err != nil {
	        if err != io.EOF {
	          fmt.Println(err)
	        }
	        break
	    }
	    hashsum := sha256.Sum256(buffer[:bytesread])
	    gossiper.SafeFileIndex.mux.Lock()
	    gossiper.SafeFileIndex.FileIndex[hashsum] = buffer[:bytesread]
	    gossiper.SafeFileIndex.mux.Unlock()
	    metafile = append(metafile, hashsum[:]...)
	}
	metafile_hash := sha256.Sum256(metafile)
	gossiper.SafeFileIndex.mux.Lock()
	gossiper.SafeFileIndex.FileIndex[metafile_hash] = metafile
	gossiper.SafeFileIndex.mux.Unlock()
	number_of_chunks := len(metafile) / 32
	gossiper.SafeFileData.mux.Lock()
	gossiper.SafeFileData.FileData[path] = &File{metafile_hash, uint64(number_of_chunks), uint64(number_of_chunks)}
	gossiper.SafeFileData.mux.Unlock()
	fmt.Println("HASH OF INDEXED", hex.EncodeToString(metafile_hash[:]))
}

func get_chunk_by_hash(gossiper *Gossiper, dest string, hash [32]byte) {
	data_request := DataRequest{gossiper.Nodename, dest, 9, hash[:]}
	packet := GossipPacket{DataRequest: &data_request}
	for {
		gossiper.SafeFileIndex.mux.Lock()
		_, ok := gossiper.SafeFileIndex.FileIndex[hash] 
		if ok == true {
			gossiper.SafeFileIndex.mux.Unlock()
			return
		}
		gossiper.SafeFileIndex.mux.Unlock()
		gossiper.RoutingTable.mux.Lock()
	    sendMessage(&packet, gossiper.RoutingTable.RouteTable[dest], gossiper.PeerConn)
	    gossiper.RoutingTable.mux.Unlock()
		time.Sleep(time.Duration(file_timeout) * time.Second)
	}
}

func download_file(gossiper *Gossiper, destination string, hash []byte, save_as string) {
	fmt.Println("REQUESTING", save_as, "FROM", destination, "HASH", hex.EncodeToString(hash))
	ex, err := os.Executable()
    if err != nil {
        fmt.Println(err)
    }
    exPath := filepath.Dir(ex)
    abs_path := filepath.Join(exPath, "_Downloads", save_as)
	metahashvalue := [32]byte{}
	copy(metahashvalue[:], hash)
	dest := destination
	if destination == "" {
		gossiper.SafeSearchResults.mux.Lock()
		randInd1 := rand.Intn(len(gossiper.SafeSearchResults.SearchResults[metahashvalue]))
		randInd2 := rand.Intn(len(gossiper.SafeSearchResults.SearchResults[metahashvalue][randInd1]))
		dest = gossiper.SafeSearchResults.SearchResults[metahashvalue][randInd1][randInd2]
		gossiper.SafeSearchResults.mux.Unlock()
	}
   	fmt.Println("DOWNLOADING metafile of", save_as, "from", dest)
	go get_chunk_by_hash(gossiper, dest, metahashvalue)
	for {
		gossiper.SafeFileIndex.mux.Lock()
		_, ok := gossiper.SafeFileIndex.FileIndex[metahashvalue] 
		if ok == true {
			gossiper.SafeFileIndex.mux.Unlock()
			break
		}
		gossiper.SafeFileIndex.mux.Unlock()
	}
	gossiper.SafeFileIndex.mux.Lock()
	metafile := gossiper.SafeFileIndex.FileIndex[metahashvalue]
	gossiper.SafeFileIndex.mux.Unlock()
	file := []byte{}
	number_of_chunks := len(metafile) / 32
	gossiper.SafeFileData.mux.Lock()
	gossiper.SafeFileData.FileData[save_as] = &File{metahashvalue, uint64(0), uint64(number_of_chunks)}
	gossiper.SafeFileData.mux.Unlock()
	hashvalue := [32]byte{}
	for chunk := 0; chunk < number_of_chunks; chunk++ {
		copy(hashvalue[:], metafile[chunk*32:(chunk+1)*32])
		if destination == "" {
			gossiper.SafeSearchResults.mux.Lock()
			randInd := rand.Intn(len(gossiper.SafeSearchResults.SearchResults[metahashvalue][chunk]))
			dest = gossiper.SafeSearchResults.SearchResults[metahashvalue][chunk][randInd]
			gossiper.SafeSearchResults.mux.Unlock()
		}
		fmt.Println("DOWNLOADING", save_as, "chunk", chunk+1, "from", dest)
		go get_chunk_by_hash(gossiper, dest, hashvalue)
		for {
			gossiper.SafeFileIndex.mux.Lock()
			_, ok := gossiper.SafeFileIndex.FileIndex[hashvalue] 
			if ok == true {
				gossiper.SafeFileIndex.mux.Unlock()
				break
			}
			gossiper.SafeFileIndex.mux.Unlock()
		}
		gossiper.SafeFileData.mux.Lock()
		gossiper.SafeFileData.FileData[save_as].LastChunk = uint64(chunk+1)
		gossiper.SafeFileData.mux.Unlock()
		gossiper.SafeFileIndex.mux.Lock()
		file = append(file, gossiper.SafeFileIndex.FileIndex[hashvalue]...)
		gossiper.SafeFileIndex.mux.Unlock()
	}
	outfile, err := os.Create(abs_path)
	if err != nil {
	    fmt.Println(err)
	    return
	}
	defer outfile.Close()
	_, err = outfile.Write(file)
	if err != nil {
	    fmt.Println(err)
	    return
	}
	fmt.Println("RECONSTRUCTED file", save_as)
}
