package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"encoding/hex"
	"time"
)

func (b *Block) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	h.Write(b.Nonce[:])
	binary.Write(h, binary.LittleEndian, uint32(len(b.Transactions)))
	for _, t := range b.Transactions {
		th := t.Hash()
		h.Write(th[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h,binary.LittleEndian,
	uint32(len(t.File.Name)))
	h.Write([]byte(t.File.Name))
	h.Write(t.File.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}

func countZeros(hash [32]byte) (int) {
	num := 0
	for _, v := range hash {
	    if v != 0 {
	        break
	    }
	    num += 1
	}
	return num
}

func mine(gossiper *Gossiper) {
	start_time := time.Now().UnixNano()
	prevHash := [32]byte{}
	for {
		gossiper.SafeTxPool.mux.Lock()
		gossiper.SafeBlocksRegister.mux.Lock()
		my_chain_length := gossiper.SafeBlocksRegister.TailMap[prevHash]
		biggest_length := 0
		biggest_hash := [32]byte{}
		for tail_hash, tail_len := range gossiper.SafeBlocksRegister.TailMap {
			if tail_len > biggest_length {
				biggest_length = tail_len
				biggest_hash = tail_hash
			}
		}
		if(my_chain_length < biggest_length) {
			processFork(gossiper, prevHash, biggest_hash)
			prevHash = biggest_hash
			printChain(gossiper, biggest_hash)
			index := -1 
			for i, fork_hash := range gossiper.SafeBlocksRegister.NewForks {
				if(gossiper.SafeBlocksRegister.ForkPointMap[fork_hash] == gossiper.SafeBlocksRegister.ForkPointMap[prevHash]) {
					index = i
					break
				}
			}
			if index != -1 {
				copy(gossiper.SafeBlocksRegister.NewForks[index:], gossiper.SafeBlocksRegister.NewForks[index+1:])
		        gossiper.SafeBlocksRegister.NewForks = gossiper.SafeBlocksRegister.NewForks[:len(gossiper.SafeBlocksRegister.NewForks)-1]
		    }
		}
		for _, fork_hash := range gossiper.SafeBlocksRegister.NewForks {
			print_hash := gossiper.SafeBlocksRegister.ForkPointMap[fork_hash]
			fmt.Println("FORK-SHORTER", hex.EncodeToString(print_hash[:]))
		}
		gossiper.SafeBlocksRegister.NewForks = [][32]byte{}
		gossiper.SafeBlocksRegister.mux.Unlock()
		if(len(gossiper.SafeTxPool.TxPool) == 0) {
			gossiper.SafeTxPool.mux.Unlock()
			start_time = time.Now().UnixNano()
			continue;
		}
		gossiper.SafeTxPool.mux.Unlock()

		nonce := [32]byte{}
		rand.Read(nonce[:])
		gossiper.SafeTxPool.mux.Lock()
		b := Block{prevHash, nonce, gossiper.SafeTxPool.TxPool}
		hash := b.Hash()
		if (countZeros(hash) >= 2) {
			prevHash = hash
			gossiper.SafeTxPool.TxPool = []TxPublish{}
			fmt.Println("FOUND-BLOCK", hex.EncodeToString(hash[:]))

			gossiper.SafeBlocksRegister.mux.Lock()
			gossiper.SafeBlocksRegister.BlocksRegister[hash] = b
			for _, t := range b.Transactions {
				gossiper.SafeBlocksRegister.NameIndex[t.File.Name] = true
			}
			forkInsertion(gossiper, b, false)
			printChain(gossiper, hash)
			gossiper.SafeBlocksRegister.mux.Unlock()
			gossiper.SafeTxPool.mux.Unlock()

			if b.PrevHash == [32]byte{} {
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(2*time.Duration(time.Now().UnixNano() - start_time) * time.Nanosecond)
			}
			
			gossiper.SafePeersList.mux.Lock()
			bp := BlockPublish{b, 19}
			for _, v := range gossiper.SafePeersList.PeersList {
				sendMessage(&GossipPacket{BlockPublish: &bp}, v, gossiper.PeerConn)
			}
			gossiper.SafePeersList.mux.Unlock()
			start_time = time.Now().UnixNano()
		} else {
			gossiper.SafeTxPool.mux.Unlock()
		}
	}
}

func isTxValid(gossiper *Gossiper, tx TxPublish) (bool) {
	gossiper.SafeBlocksRegister.mux.Lock()
	_, ok := gossiper.SafeBlocksRegister.NameIndex[tx.File.Name] 
	if ok == true {
		gossiper.SafeBlocksRegister.mux.Unlock()
		return false
	}
	gossiper.SafeBlocksRegister.mux.Unlock()
	good := true
    for _, t := range gossiper.SafeTxPool.TxPool {
    	if t.File.Name == tx.File.Name {
    		good = false
    		break
    	}
    }
    return good
}

func forkInsertion(gossiper *Gossiper, block Block, inlist bool) {
	fork := true
	for k, _ := range gossiper.SafeBlocksRegister.TailMap {
		if block.PrevHash == k {
			value := gossiper.SafeBlocksRegister.TailMap[k]
			delete(gossiper.SafeBlocksRegister.TailMap, k)
			gossiper.SafeBlocksRegister.TailMap[block.Hash()] = value + 1
			gossiper.SafeBlocksRegister.ForkPointMap[block.Hash()] = gossiper.SafeBlocksRegister.ForkPointMap[block.PrevHash]
			fork = false
			break
		}
	}
	if fork == true {
		lenght := 1
		current_block := block
		for {
			prev_block, ok := gossiper.SafeBlocksRegister.BlocksRegister[current_block.PrevHash]
			if ok == false {
				break
			}
			lenght += 1
			current_block = prev_block
		}
		gossiper.SafeBlocksRegister.TailMap[block.Hash()] = lenght
		gossiper.SafeBlocksRegister.ForkPointMap[block.Hash()] = block.PrevHash
		if inlist {
			gossiper.SafeBlocksRegister.NewForks = append(gossiper.SafeBlocksRegister.NewForks, block.Hash())
		}
	}
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func findCommonPoint(gossiper *Gossiper, old_chain_hash [32]byte, new_chain_hash [32]byte) ([32]byte) {
	common_point := [32]byte{}
	list_a := [][32]byte{}
	block_a := gossiper.SafeBlocksRegister.BlocksRegister[old_chain_hash]
	for {
		list_a = append([][32]byte{gossiper.SafeBlocksRegister.ForkPointMap[block_a.Hash()]}, list_a...)
		prev_block, ok := gossiper.SafeBlocksRegister.BlocksRegister[gossiper.SafeBlocksRegister.ForkPointMap[block_a.Hash()]]
		if ok == false {
			break
		}
		block_a = prev_block
	}

	list_b := [][32]byte{}
	block_b := gossiper.SafeBlocksRegister.BlocksRegister[new_chain_hash]
	for {
		list_b = append([][32]byte{gossiper.SafeBlocksRegister.ForkPointMap[block_b.Hash()]}, list_b...)
		prev_block, ok := gossiper.SafeBlocksRegister.BlocksRegister[gossiper.SafeBlocksRegister.ForkPointMap[block_b.Hash()]]
		if ok == false {
			break
		}
		block_b = prev_block
	}

	min_length := min(len(list_a), len(list_b))
	not_common := -1
	for i:=0; i<min_length; i++ {
		if list_a[i] != list_b[i] {
			not_common = i
			break
		}
	}

	if not_common == -1 {
		common_point = list_a[min_length-1]
		if len(list_a) > len(list_b) {
			block_b := gossiper.SafeBlocksRegister.BlocksRegister[new_chain_hash]
			new_common_point := list_a[min_length]
			seen := false
			for {
				if block_b.Hash() == new_common_point {
					seen = true
					break
				}
				if block_b.PrevHash == common_point {
					break;
				}
				block_b = gossiper.SafeBlocksRegister.BlocksRegister[block_b.PrevHash]
			}
			if seen {
				common_point = new_common_point
			}
		}
		if len(list_a) < len(list_b) {
			block_a := gossiper.SafeBlocksRegister.BlocksRegister[old_chain_hash]
			new_common_point := list_b[min_length]
			seen := false
			for {
				if block_a.Hash() == new_common_point {
					seen = true
					break
				}
				if block_a.PrevHash == common_point {
					break;
				}
				block_a = gossiper.SafeBlocksRegister.BlocksRegister[block_a.PrevHash]
			}
			if seen {
				common_point = new_common_point
			}
		}
	} else {
		block_a = gossiper.SafeBlocksRegister.BlocksRegister[list_a[not_common]]
		block_b = gossiper.SafeBlocksRegister.BlocksRegister[list_b[not_common]]
		common_fork_point := list_b[not_common-1]
		found_b := false
		found_a := false
		for {
			if block_a.Hash() == block_b.Hash() {
				found_b = true
				break
			}
			if block_a.PrevHash == common_fork_point {
				break;
			}
			block_a = gossiper.SafeBlocksRegister.BlocksRegister[block_a.PrevHash]
		}
		block_a = gossiper.SafeBlocksRegister.BlocksRegister[list_a[not_common]]

		for {
			if block_a.Hash() == block_b.Hash() {
				found_a = true
				break
			}
			if block_b.PrevHash == common_fork_point {
				break;
			}
			block_b = gossiper.SafeBlocksRegister.BlocksRegister[block_b.PrevHash]
		}
		block_b = gossiper.SafeBlocksRegister.BlocksRegister[list_b[not_common]]

		if(found_a && !found_b) {
			common_point = block_a.Hash()
		}
		if(!found_a && found_b) {
			common_point = block_b.Hash()
		}
		if(!found_a && !found_b) {
			common_point = common_fork_point
		}
	}
	return common_point
}

func processFork(gossiper *Gossiper, old_chain_hash [32]byte, new_chain_hash [32]byte) {
	common_point := findCommonPoint(gossiper, old_chain_hash, new_chain_hash)
	//fmt.Println("COMMON POINT", hex.EncodeToString(common_point[:]))
	not_rewind := false
	current_block := gossiper.SafeBlocksRegister.BlocksRegister[new_chain_hash]
	for {
		if current_block.Hash() == old_chain_hash {
			not_rewind = true
			common_point = old_chain_hash
			break
		}
		if current_block.PrevHash == common_point {
			break;
		}
		current_block = gossiper.SafeBlocksRegister.BlocksRegister[current_block.PrevHash]
	}
	current_block = gossiper.SafeBlocksRegister.BlocksRegister[old_chain_hash]
	counter := 0
	for {
		if not_rewind || current_block.Hash() == common_point {
			break
		}
		counter += 1
		for _, tx := range current_block.Transactions {
			delete(gossiper.SafeBlocksRegister.NameIndex, tx.File.Name)
			//fmt.Println("ADD IN REWIND", tx.File.Name)
			gossiper.SafeTxPool.TxPool = append(gossiper.SafeTxPool.TxPool, tx)
		}
		if current_block.PrevHash == common_point {
			break
		}
		current_block = gossiper.SafeBlocksRegister.BlocksRegister[current_block.PrevHash]
	}
	if counter > 0 {
		fmt.Println("FORK-LONGER", "rewind", counter, "blocks")
	}

	current_block = gossiper.SafeBlocksRegister.BlocksRegister[new_chain_hash]
	for {
		for _, tx := range current_block.Transactions {
			gossiper.SafeBlocksRegister.NameIndex[tx.File.Name] = true
			index := -1
			for i, t := range gossiper.SafeTxPool.TxPool {
		    	if t.File.Name == tx.File.Name {
		    		index = i
		    		break
		    	}
		    }
		    if index != -1 {
		    	//fmt.Println("REMOVE IN REWIND", gossiper.SafeTxPool.TxPool[index].File.Name)
		    	copy(gossiper.SafeTxPool.TxPool[index:], gossiper.SafeTxPool.TxPool[index+1:])
	            gossiper.SafeTxPool.TxPool = gossiper.SafeTxPool.TxPool[:len(gossiper.SafeTxPool.TxPool)-1]
		    }
		}
		if current_block.PrevHash == common_point {
			break
		}
		current_block = gossiper.SafeBlocksRegister.BlocksRegister[current_block.PrevHash]
	}
}

func printChain(gossiper *Gossiper, hash [32]byte) {
	message := "CHAIN"
	current_block := gossiper.SafeBlocksRegister.BlocksRegister[hash]
	for {
		hash := current_block.Hash()
		message += " "+hex.EncodeToString(hash[:])+":"+hex.EncodeToString(current_block.PrevHash[:])
		first := true
		for _, tx := range current_block.Transactions {
			if first {
				message += ":"
				first = false
			} else {
				message += ","
			}
			message += tx.File.Name
		}
		prev_block, ok := gossiper.SafeBlocksRegister.BlocksRegister[current_block.PrevHash]
		if ok == false {
			break
		}
		current_block = prev_block
	}
	fmt.Println(message)
}
