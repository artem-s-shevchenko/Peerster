package main

import (
	"fmt"
	"time"
	"math/rand"
	"encoding/hex"
	"sort"
	"strings"
	"regexp"
)

func uint64ToString(a []uint64) string {
    b := ""
    for _, v := range a {
        if len(b) > 0 {
            b += ","
        }
        b += fmt.Sprint(v)
    }
    return b
}

func processSearchResults(gossiper *Gossiper, dec *GossipPacket) {
	for _, result := range dec.SearchReply.Results {
    	fmt.Println("FOUND match", result.FileName, "at node", dec.SearchReply.Origin, "metafile="+hex.EncodeToString(result.MetafileHash), 
    		"chunks="+uint64ToString(result.ChunkMap))
    	hashvalue := [32]byte{}
		copy(hashvalue[:], result.MetafileHash)
		fmt.Println(hashvalue)
		number_of_chunks := result.ChunkCount
		gossiper.SafeSearchResults.mux.Lock()
		_, ok := gossiper.SafeSearchResults.SearchResults[hashvalue]
		if ok == false {
			for chunk := uint64(0); chunk < number_of_chunks; chunk++ {
				gossiper.SafeSearchResults.SearchResults[hashvalue] = append(gossiper.SafeSearchResults.SearchResults[hashvalue], []string{})
			}
		}
		for _, chunk_index := range result.ChunkMap {
			stop := false
			for _, v := range gossiper.SafeSearchResults.SearchResults[hashvalue][chunk_index-1] {
				if v == dec.SearchReply.Origin {
					stop = true
					break
				}
			}
			if stop == true {
				continue
			}
 			gossiper.SafeSearchResults.SearchResults[hashvalue][chunk_index-1] = append(gossiper.SafeSearchResults.SearchResults[hashvalue][chunk_index-1], 
				dec.SearchReply.Origin)
		}
		full := true
		for _, chunkOwners := range gossiper.SafeSearchResults.SearchResults[hashvalue] {
			if len(chunkOwners) == 0 {
				full = false
				break
			}
		}
		gossiper.SafeSearchResults.mux.Unlock()
		if full == true {
			gossiper.SafeKeywordResultMapping.mux.Lock()
			for key, _ := range gossiper.SafeKeywordResultMapping.KeywordResultMapping {
				keywords := strings.Split(key, ",")
				for _, k := range keywords {
			    	reg_exp := regexp.MustCompile("^.*" + k + ".*$")
			    	if reg_exp.MatchString(result.FileName) {
			    		stop := false
						for _, v := range gossiper.SafeKeywordResultMapping.KeywordResultMapping[key] {
							if v == hashvalue {
								stop = true
								break
							}
						}
						if stop == true {
							continue
						}
			    		gossiper.SafeKeywordResultMapping.KeywordResultMapping[key] = append(gossiper.SafeKeywordResultMapping.KeywordResultMapping[key], hashvalue)
			    	}
			    }
			}
			gossiper.SafeKeywordResultMapping.mux.Unlock()
		}
	}
}


func search(gossiper *Gossiper, budget uint64, keywords []string, increase bool) {
	sort.Strings(keywords)
	key := strings.Join(keywords,",")
	gossiper.SafeKeywordResultMapping.mux.Lock()
	_, ok := gossiper.SafeKeywordResultMapping.KeywordResultMapping[key]
	if ok == false {
		gossiper.SafeKeywordResultMapping.KeywordResultMapping[key] = [][32]byte{}
	}
	gossiper.SafeKeywordResultMapping.mux.Unlock()
	req := SearchRequest{gossiper.Nodename, budget, keywords}
	for {
		resendSearchRequest(gossiper, &GossipPacket{SearchRequest: &req}, "")
		if increase == false {
			fmt.Println("SEARCH FINISHED")
			break
		}
		gossiper.SafeKeywordResultMapping.mux.Lock()
		if len(gossiper.SafeKeywordResultMapping.KeywordResultMapping[key])>=2 {
			gossiper.SafeKeywordResultMapping.mux.Unlock()
			fmt.Println("SEARCH FINISHED")
			break
		}
		gossiper.SafeKeywordResultMapping.mux.Unlock()
		req.Budget = 2*(req.Budget+1)
		if req.Budget>=32 {
			fmt.Println("SEARCH FINISHED")
			break
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func resendSearchRequest(gossiper *Gossiper, dec *GossipPacket, not_send string) {
	dec.SearchRequest.Budget -= 1
	if dec.SearchRequest.Budget > 0 {
		requests := []SearchRequest{}
		gossiper.SafePeersList.mux.Lock()
		if len(gossiper.SafePeersList.PeersList) > 0 {
			equal_distribution := dec.SearchRequest.Budget/uint64(len(gossiper.SafePeersList.PeersList))
			for i:=0; i<len(gossiper.SafePeersList.PeersList); i++ {
				requests = append(requests, SearchRequest{dec.SearchRequest.Origin, equal_distribution, dec.SearchRequest.Keywords})
			}
			rest_of_budget := dec.SearchRequest.Budget % uint64(len(gossiper.SafePeersList.PeersList))
			for rest_of_budget > 0 {
				randInd := rand.Intn(len(gossiper.SafePeersList.PeersList))
				for requests[randInd].Budget > equal_distribution {
		            randInd = rand.Intn(len(gossiper.SafePeersList.PeersList))
		        }
		        requests[randInd].Budget += 1
		        rest_of_budget--;
		    }
		    for i, req := range requests {
		    	if req.Budget > 0 && gossiper.SafePeersList.PeersList[i] != not_send {
		    		sendMessage(&GossipPacket{SearchRequest: &req}, gossiper.SafePeersList.PeersList[i], gossiper.PeerConn)
		    	}
		    }
		}
    	gossiper.SafePeersList.mux.Unlock()
	}
}

func detectDuplicateRequest(gossiper *Gossiper, dec *GossipPacket) bool {
	if dec.SearchRequest.Origin == gossiper.Nodename {
		return false
	}
	key := strings.Join(dec.SearchRequest.Keywords,",")
	current_time := time.Now().UnixNano()
	if(current_time - gossiper.SearchRequestTime[key] > 5*1e8) {
		return false
	}
	gossiper.SearchRequestTime[key] = current_time
	return true
}
