#!/usr/local/bin/bash

# Define variables
DEBUG=false
HELP=false

numberOfPeers=10
maxNumberOfMessagesPerPeer=2
maxNumberPrivateMessagesPerPeer=1
UIPort=12345
minimumGossipPort=5000
maxNumberOfFiles=3
maxNumberOfChunksPerFile=10
RTimer=10
TimeToWait=20

TestRouting=false
TestPrivateMessages=false
TestFileIndexing=false
TestFileSharing=false
TestFile=false
TestSearch=false
TestBlockchain=false
AllowWarning=true

# Handle command-line arguments
while [[ $# -gt 0 ]]
do
    key="$1"

    case $key in
        -v|--verbose)
            echo "Mode debug"
            DEBUG=true
            ;;
        -h|--help)
            HELP=true
            ;;
        -p|--number-peers)
            shift
            numberOfPeers="$1"
            ;;
        -u|--ui-port)
            shift
            UIPort="$1"
            ;;
        -g|--gossip-port)
            shift
            minimumGossipPort="$1"
            ;;
        -r|--route-timer)
            shift
            RTimer="$1"
            TestRouting=true
            ;;
        --test-routing)
            TestRouting=true
            ;;
        -f|--file-sharing)
            TestFileSharing=true
            TestFile=true
            ;;
        -i|--file-indexing)
            TestFileIndexing=true
            TestFile=true
            ;;
        -m|--private-messages)
            TestPrivateMessages=true
            ;;
        -w|--disable-warnings)
            AllowWarning=false
            ;;
        -a|--all)
            TestRouting=true
            TestPrivateMessages=true
            TestFileIndexing=true
            TestFileSharing=true
            TestFile=true
            TestSearch=true
            TestBlockchain=true
            ;;
        -t|--wait-time)
            shift
            TimeToWait="$1"
            ;;
        -s|--search)
            TestFileIndexing=true
            TestSearch=true
            TestFile=true
            ;;
        -b|--blockchain)
            TestFile=true
            TestFileIndexing=true
            TestBlockchain=true
            ;;
        *)
            # unknown option
            ;;
    esac
    shift
done

# Define colors for our outputs
BLACK='\033[0;30m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
WHITE='\033[0;97m'
NC='\033[0m'
WARNINGCOLOR=$YELLOW
if [[ $AllowWarning = false ]]
then
    WARNINGCOLOR=$RED
fi

failed=false
warning=false

check_failed() {
    if [[ $failed = true ]]
    then
        echo -e "${RED}***FAILED***${NC}"
        failed=false
    else
        echo -e "${GREEN}***PASSED***${NC}"
    fi
}

check_warning() {
    if [[ $warning = true ]]
    then
        if [[ $AllowWarning = true ]]
        then
            echo -e "${YELLOW}***   CHECK YOUR IMPLEMENTATION   ***${NC}"
            echo -e "${YELLOW}***SOMETHING MIGHT HAVE GONE WRONG***${NC}"
        else
            echo -e "${RED}***FAILED***${NC}"
        fi
        warning=false
    else
        echo -e "${GREEN}***PASSED***${NC}"
    fi
}

if [[ $HELP = true ]]
then
    echo "usage: ./test_generic.sh [ OPTIONS ]

      -h | --help               Display this help message
      -v | --verbose            Increase the details of the displayed messages
      -p | --number-peers       Specify the number of peers to launch. Default is 10.
      -u | --ui-port            Specify the first UI port to use. Default is 12345.
      -g | --gossip-port        Specify the first gossip port to use. Default is 5000.
      -r | --route-timer        Define the value for the -rtimer flag. Implies testing the routing.
                                Default is 0.
      -a | --all                Enable all the checks
      --test-routing            Test the routing part.
      -i | --file-indexing      Test the file indexing part.
      -f | --file-sharing       Test the file sharing part.
      -s | --search             Test the search part.
      -m | --private-messages   Test the private messages part.
      -w | --disable-warnings   Trigger failure instead of warning when a private message or a
                                file-related message doesn't reach its destination.
      -t | --wait-time          Time to wait for processes to communicate, before checking the output files.
    "
else
    if [[ $numberOfPeers -le 1 ]]
    then
        echo "${RED}******** You need at least two processes to communicate ************"
        exit 0
    fi
    # Build server then client
    go build
    cd client
    go build
    cd ..
    # Interrupt all the processes, in case any of them is still running
    pkill -f Peerster
    rm *.out

    outputFiles=()
    UIPorts=()

    secondSender=2
    thirdSender=4
    messagesSentThird=0

    firstPrivateSender=1
    firstPrivateDest=5
    secondPrivateSender=7
    secondPrivateDest=0
    thirdPrivateSender=1
    thirdPrivateDest=0
    privateSentThird=0

    if [[ $numberOfPeers -lt 8 ]]
    then
        secondPrivateSender=2
        if [[ $numberOfPeers -lt 6 ]]
        then
            firstPrivateDest=2
            if [[ $numberOfPeers -lt 5 ]]
            then
                thirdSender=1
                if [[ $numberOfPeers -lt 3 ]]
                then
                    secondSender=1
                    thirdSender=0
                    messagesSentThird=2
                    maxNumberOfMessagesPerPeer=3

                    firstPrivateDest=0
                    secondPrivateSender=0
                    secondPrivateDest=1
                    privateSentThird=1
                    maxNumberPrivateMessagesPerPeer=2
                fi
            fi
        fi
    fi

    declare -A messages
    messages[0,0]=Weather_is_clear
    messages[0,1]=No_clouds_really
    messages[$secondSender,0]=Winter_is_coming
    messages[$secondSender,1]=Let\'s_go_skiing
    messages[$thirdSender,$messagesSentThird]=Is_anybody_here?

    if [[ $TestPrivateMessages = true ]]
    then
        declare -A private_messages
        private_messages[$firstPrivateSender,$firstPrivateDest,0]=Night_is_falling
        private_messages[$secondPrivateSender,$secondPrivateDest,0]=Sun_is_rising
        private_messages[$thirdPrivateSender,$thirdPrivateDest,$privateSentThird]=Go_to_bed
    fi

    if [[ $TestFile = true ]]
    then
        declare -A files
        # create a multi-dimensionnal array for files:
        # - first dimension is the node indexing it
        # - second is the node that will ask for it (if different from the first)
        # - third is the number of the file (we want to be able to have several files at each node)
        # - fourth is 0 for the filename, 1 for the file **metahash**
        files[0,1,0,0]="test"
        files[0,1,0,1]="bc62d4b80d9e36da29c16c5d4d9f11731f36052c72401a76c23c0fb5a9b74423"
        files[0,0,1,0]="local"
        files[0,0,1,1]="097f8ff321396fe5768f66065f29edfdc7ea350712f1c6fc9daf337ec1ccc7ff"
        files[1,1,1,0]="neighbor"
        files[1,1,1,1]="bcf98c067da476fb0f734ef433768a7441433d6ca2b64309466a7315e485402c"
        files[1,1,2,0]="image"
        files[1,1,2,1]="d3e9c471195b747bed5987c56d3ba94919de51960ca8af8cf58174200764f9df"
        files[1,1,2,2]="d"
        files[1,1,2,3]="d"
        files[1,1,2,4]="d"
        files[1,1,2,5]="d"
        files[1,1,2,6]="d"
        files[1,1,2,7]="d"
        if [[ $numberOfPeers -gt 3 ]]
        then
            files[2,2,1,0]="indirect"
            files[2,2,1,1]="96faf09646bd8a08ee8af9498f1a94bb03192382bcab1d59b7660cee4f8a618f"
        fi
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            files[$i,$i,0,0]="global"
            files[$i,$i,0,1]="860b99dabed09a242b5ef68f72464acb4ad7b37b596826808db71d4d21c30655"
        done
    fi

    name='A'

    # Launch peers as a circle
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        outFileName="$name.out"
        peerPort=$(($(($i + 1)) % $numberOfPeers + $minimumGossipPort))
        gossipPort=$(($i + $minimumGossipPort))
        peer="127.0.0.1:$peerPort"
        gossipAddr="127.0.0.1:$gossipPort"
        if [[ $TestRouting = false ]]
        then
            if [[ $DEBUG = true ]]
            then
                echo "./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer > $outFileName &"
            fi
            ./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer > $outFileName &
        else
            if [[ $DEBUG = true ]]
            then
                echo "./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer -rtimer=$RTimer > $outFileName &"
            fi
            ./Peerster -UIPort=$UIPort -gossipAddr=$gossipAddr -name=$name -peers=$peer -rtimer=$RTimer > $outFileName &
        fi
        outputFiles+=("$outFileName")
        if [[ "$DEBUG" == "true" ]] ; then
            echo "$name running at UIPort $UIPort and gossipPort $gossipPort"
        fi
        UIPorts+=($UIPort)
        UIPort=$(($UIPort+1))
        name=$(echo "$name" | tr "A-Y" "B-Z")
    done

    sleep 3
    # Send messages
    ./client/client -UIPort=${UIPorts[0]} -msg="${messages[0,0]}"
    ./client/client -UIPort=${UIPorts[$secondSender]} -msg="${messages[$secondSender,0]}"
    sleep 5
    if [[ $TestPrivateMessages = true ]]
    then
        # Private message
        ./client/client -UIPort=${UIPorts[$firstPrivateSender]} -msg="${private_messages[$firstPrivateSender,$firstPrivateDest,0]}" -dest="${outputFiles[$firstPrivateDest]:0:1}"
    fi
    ./client/client -UIPort=${UIPorts[0]} -msg="${messages[0,1]}"
    sleep 3
    ./client/client -UIPort=${UIPorts[$secondSender]} -msg="${messages[$secondSender,1]}"
    ./client/client -UIPort=${UIPorts[$thirdSender]} -msg="${messages[$thirdSender,$messagesSentThird]}"
    sleep 3
    if [[ $TestPrivateMessages = true ]]
    then
        ./client/client -UIPort=${UIPorts[$secondPrivateSender]} -msg="${private_messages[$secondPrivateSender,$secondPrivateDest,0]}" -dest="${outputFiles[$secondPrivateDest]:0:1}"
        ./client/client -UIPort=${UIPorts[$thirdPrivateSender]} -msg="${private_messages[$thirdPrivateSender,$thirdPrivateDest,$privateSentThird]}" -dest="${outputFiles[$thirdPrivateDest]:0:1}"
    fi
    if [[ $TestFile = true ]]
    then
        # IndexFile
        ./client/client -UIPort=${UIPorts[0]} -file="${files[0,1,0,0]}"

        # Wait for the node to have indexed the file
        sleep 10

        if [[ $TestSearch = true ]] || [[ $TestBlockchain = true ]]
        then
            # For the search or the blockchain, index all files destined to self
            for i in `seq 0 $(($numberOfPeers - 1))`
            do
                for j in `seq 0 $(($maxNumberOfFiles - 1))`
                do
                    if [[ ${files[$i,$i,$j,0]} != "" ]]
                    then
                        ./client/client -UIPort=${UIPorts[$i]} -file="${files[$i,$i,$j,0]}"
                        if [[ $TestBlockchain = true ]]
                        then
                            sleep 5
                        fi
                    fi
                done
            done
        fi

        if [[ $TestFileSharing ]]
        then
            # Request the file
            ./client/client -UIPort=${UIPorts[1]} -file="${files[0,1,0,0]}2" -request="${files[0,1,0,1]}" -dest="${outputFiles[0]:0:1}"
        fi

        keywords="loc,cal neigh,bor ind,irect glo,bal im"
        if [[ $TestSearch = true ]]
        then
            for i in $keywords
            do
                ./client/client -UIPort=${UIPorts[0]} -keywords="$i" -budget=64
            done
            sleep 15
            if [[ $TestFileSharing ]]
            then
                ./client/client -UIPort=${UIPorts[0]} -file="${files[1,1,2,0]}2" -request="${files[1,1,2,1]}"
            fi
        fi
    fi
    # Wait for the nodes to communicate with one another
    sleep $TimeToWait
    # Interrupt all the processes
    pkill -f Peerster

    #######################
    # Testing correctness #
    #######################

    # Client messages
    failed=false
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK that client messages arrived${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        for j in `seq 0 $(($maxNumberOfMessagesPerPeer - 1))`
        do
            if [[ ${messages[$i,$j]} != "" ]] && !(grep -q "CLIENT MESSAGE ${messages[$i,$j]}" "${outputFiles[$i]}")
            then
                failed=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${RED}CLIENT MESSAGE ${messages[$i,$j]} not present in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
    done
    check_failed

    # Client private messages
    if [[ $TestPrivateMessages = true ]]
    then
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK that client private messages arrived${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            for j in `seq 0 $(($numberOfPeers - 1))`
            do
                for k in `seq 0 $(($maxNumberPrivateMessagesPerPeer - 1))`
                do
                    line="SENDING PRIVATE MESSAGE ${private_messages[$i,$j,$k]} TO ${outputFiles[$j]:0:1}"
                    if [[ ${private_messages[$i,$j,$k]} != "" ]] && !(grep -q "$line" "${outputFiles[$i]}")
                    then
                        failed=true
                        if [[ "$DEBUG" == "true" ]] ; then
                            echo -e "${RED}$line not present in ${outputFiles[$i]}${NC}"
                        fi
                    fi
                done
            done
        done
        check_failed
    fi

    if [[ $TestFileIndexing = true ]]
    then
        # Client file indexing messages
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK that client file-indexing messages arrived${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            for j in `seq 0 $(($numberOfPeers - 1))`
            do
                for k in `seq 0 $(($maxNumberOfFiles - 1))`
                do
                    line="INDEXING ${files[$i,$j,$k,0]}"
                    if [[ $i != $j ]] && [[ ${files[$i,$j,$k,0]} != "" ]] && !(grep -q "$line" "${outputFiles[$i]}")
                    then
                        failed=true
                        if [[ "$DEBUG" == "true" ]] ; then
                            echo -e "${RED}$line not present in ${outputFiles[$i]}${NC}"
                        fi
                    fi
                done
            done
        done
        check_failed
    fi

    if [[ $TestFileSharing = true ]]
    then
        # Client file requesting messages
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK that client file-requesting messages arrived${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            for j in `seq 0 $(($numberOfPeers - 1))`
            do
                for k in `seq 0 $(($maxNumberOfFiles - 1))`
                do
                    if [[ "$i" -ne "$j" ]] && [[ ${files[$j,$i,$k,0]} != "" ]]
                    then
                        line="REQUESTING ${files[$j,$i,$k,0]}2 FROM ${outputFiles[$j]:0:1} HASH ${files[$j,$i,$k,1]}"
                        if !(grep -q "$line" "${outputFiles[$i]}")
                        then
                            failed=true
                            if [[ "$DEBUG" == "true" ]] ; then
                                echo -e "${RED}$line not present in ${outputFiles[$i]}${NC}"
                            fi
                        fi
                    fi
                done
            done
        done
        check_failed
    fi

    # Rumor messages
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK rumor messages ${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        # for each peer, check that all messages have been seen as rumor
        for j in `seq 0 $(($numberOfPeers - 1))`
        do
            for k in `seq 0 $(($maxNumberOfMessagesPerPeer - 1))`
            do
                if [[ ${messages[$j,$k]} != "" ]] && [[ $i != $j ]]
                then
                    line="RUMOR origin ${outputFiles[$j]:0:1} from 127.0.0.1:[0-9]{4} ID [0-9]+ contents ${messages[$j,$k]}"
                    if !(grep -Eq "$line" "${outputFiles[$i]}") ; then
                        failed=true
                        if [[ $DEBUG = true ]] ; then
                            echo -e "${RED}$line not present in ${outputFiles[$i]}${NC}"
                        fi
                    fi
                fi
            done
        done
    done
    check_failed

    # Mongering
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK mongering ${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        # for each node, check that it has mongered with its two peers
        for j in "$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))" "$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))"
        do
            if !(grep -q "MONGERING with 127.0.0.1:$j" "${outputFiles[$i]}")
            then
                failed=true
                if [[ $DEBUG = true ]]
                then
                    echo -e "${RED}Node ${outputFiles[$i]:0:1} did not monger with peer ${outputFiles[$(($j - $minimumGossipPort))]:0:1}${NC}"
                    echo -e "${RED}MONGERING with 127.0.0.1:$j not present in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
    done
    check_failed

    # Check status messages
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK status messages ${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        patterns=()
        patterns+=("STATUS from 127.0.0.1:$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))"  "STATUS from 127.0.0.1:$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))")
        for j in `seq 0 $(($numberOfPeers - 1))`
        do
            for k in `seq 1 $(($maxNumberOfMessagesPerPeer - 1))`
            do
                if [[ ${messages[$j,$k]} != "" ]]
                then
                    patterns+=("peer ${outputFiles[$j]:0:1} nextID $(($k+1))")
                fi
            done
        done
        for pattern in "${patterns[@]}"
        do
            if !(grep -q "$pattern" "${outputFiles[$i]}")
            then
            failed=true
            if [[ $DEBUG = true ]]
            then
                echo -e "${RED}$pattern not present in ${outputFiles[$i]}${NC}"
            fi
        fi
        done
    done
    check_failed

    # Check flipped coins
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK flipped coin${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers - 1))`
    do
        for j in "$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))" "$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))"
        do
            if !(grep -q "FLIPPED COIN sending rumor to 127.0.0.1:$j" "${outputFiles[$i]}")
            then
                warning=true
                if [[ $DEBUG = true ]]
                then
                    echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} did not flip a coin and sent a rumor to peer ${outputFiles[$(($j - $minimumGossipPort))]:0:1}${NC}"
                fi
            fi
        done
    done
    check_warning

    # Check in sync
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK in sync${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers-1))`
    do
        for j in "$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))" "$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))"
        do
            if !(grep -q "IN SYNC WITH 127.0.0.1:$j" "${outputFiles[$i]}")
            then
                failed=true
                if [[ $DEBUG = true ]]
                then
                    echo -e "${RED}Node ${outputFiles[$i]:0:1} not in sync with peer ${outputFiles[$(($j - $minimumGossipPort))]:0:1}${NC}"
                    echo -e "${RED}IN SYNC WITH 127.0.0.1:$j not present in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
    done
    check_failed

    # Check correct peers
    if [[ $DEBUG = true ]]
    then
        echo -e "${BLUE}###CHECK correct peers${NC}"
    fi
    for i in `seq 0 $(($numberOfPeers-1))`
    do
        portBelow="$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))"
        portAbove="$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))"
        if ([[ $numberOfPeers -gt 2 ]] && !(grep -q "127.0.0.1:$portBelow,127.0.0.1:$portAbove" "${outputFiles[$i]}") &&  !(grep -q "127.0.0.1:$portAbove,127.0.0.1:$portBelow" "${outputFiles[$i]}") ) || !(grep -q "127.0.0.1:$portBelow" "${outputFiles[$i]}")
        then
            failed=true
            if [[ $DEBUG = true ]]
            then
                echo -e "${RED}Node ${outputFiles[$i]:0:1} has not the right peers${NC}"
            fi
        fi
    done
    check_failed

    if [[ $TestRouting = true ]]
    then
        # Check update routing table
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK update routing table${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers-1))`
        do
            portBelow="$(( $(($i-1+$numberOfPeers)) % $numberOfPeers + $minimumGossipPort ))"
            portAbove="$(( $(($i+1)) % $numberOfPeers + $minimumGossipPort ))"
            for j in `seq 0 $(($numberOfPeers-1))`
            do
                if [[ ${messages[$j,0]} != "" ]] && [[ $j != $i ]]
                then
                    if !(grep -q "DSDV ${outputFiles[$j]:0:1} 127.0.0.1:$portBelow" "${outputFiles[$i]}") &&  !(grep -q "DSDV ${outputFiles[$j]:0:1} 127.0.0.1:$portAbove" "${outputFiles[$i]}")
                    then
                        failed=true
                        if [[ $DEBUG = true ]]
                        then
                            echo -e "${RED}Node ${outputFiles[$i]:0:1} does not update the routing table${NC}"
                            echo -e "${RED}'DSDV ${outputFiles[$j]:0:1} 127.0.0.1:$portBelow' and 'DSDV ${outputFiles[$j]:0:1} 127.0.0.1:$portAbove' not present in ${outputFiles[$i]}${NC}"
                        fi
                    fi
                fi
            done
        done
        check_failed
    fi

    if [[ $TestPrivateMessages = true ]]
    then
        # Check private messages
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK private messages${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers-1))`
        do
            for j in `seq 0 $(($numberOfPeers-1))`
            do
                for k in `seq 0 $(($maxNumberPrivateMessagesPerPeer - 1))`
                do
                    if [[ ${private_messages[$j,$i,$k]} != "" ]]
                    then
                        if !(grep -Eq "PRIVATE origin ${outputFiles[$j]:0:1} hop-limit [0-9]+ contents ${private_messages[$j,$i,$k]}" "${outputFiles[$i]}")
                        then
                            warning=true
                            if [[ $DEBUG = true ]]
                            then
                                echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not receive private messages${NC}"
                                echo -e "${WARNINGCOLOR}'PRIVATE origin ${outputFiles[$j]:0:1} hop-limit [0-9]+ contents ${private_messages[$j,$i,$k]}' not present in ${outputFiles[$i]}${NC}"
                            fi
                        fi
                    fi
                done
            done
        done
        check_warning
    fi

    if [[ $TestFileSharing = true ]]
    then
        # Check file downloading
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK file downloading${NC}"
        fi
        for i in `seq 0 $(($numberOfPeers-1))`
        do
            for j in `seq 0 $(($numberOfPeers-1))`
            do
                for k in `seq 0 $(($maxNumberOfFiles-1))`
                do
                    # Check the start of a download
                    if [[ $i != $j ]] && [[ ${files[$j,$i,$k,0]} != "" ]]
                    then
                        if [[ $DEBUG = true ]]
                        then
                            echo -e "${CYAN}Checking download of metafile for file ${files[$j,$i,$k,0]} in output file ${outputFiles[$i]}"
                        fi
                        if !(grep -q "DOWNLOADING metafile of ${files[$j,$i,$k,0]}2 from ${outputFiles[$j]:0:1}" "${outputFiles[$i]}")
                        then
                            warning=true
                            if [[ $DEBUG = true ]]
                            then
                                echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not receive metafile of ${files[$j,$i,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                            fi
                        fi

                        if [[ $warning != true ]]
                        then
                            # Check each chunk
                            for l in `seq 1 $maxNumberOfChunksPerFile`
                            do
                                if [[ ${files[$j,$i,$k,$l]} != "" ]]
                                then
                                    if [[ $DEBUG = true ]]
                                    then
                                        echo -e "${CYAN}Checking download of chunk $l for file ${files[$j,$i,$k,0]} in output file ${outputFiles[$i]}"
                                    fi
                                    if !(grep -q "DOWNLOADING ${files[$j,$i,$k,0]}2 chunk $l from ${outputFiles[$j]:0:1}" "${outputFiles[$i]}")
                                    then
                                        warning=true
                                        if [[ $DEBUG = true ]]
                                        then
                                            echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not receive chunk $l of ${files[$j,$i,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                                        fi
                                    fi
                                fi
                            done

                            # Check the full reception of the file
                            if [[ $warning != true ]] && !(grep -q "RECONSTRUCTED file ${files[$j,$i,$k,0]}2" "${outputFiles[$i]}")
                            then
                                warning=true
                                if [[ $DEBUG = true ]]
                                then
                                    echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not reconstruct file ${files[$j,$i,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                                fi
                            fi
                            if [[ $warning != true ]] && [[ $(diff "_Downloads/${files[$j,$i,$k,0]}2" "_SharedFiles/${files[$j,$i,$k,0]}") != "" ]]
                            then
                                warning=true
                                if [[ $DEBUG = true ]]
                                then
                                    echo -e "${WARNINGCOLOR}Reconstructed file ${files[$j,$i,$k,0]} does not match the original${NC}"
                                fi
                            fi
                        fi
                    fi
                done
            done
        done
        if [[ $TestSearch = true ]]
        then
            i=0
            j=1
            k=2
            if [[ $DEBUG = true ]]
            then
                echo -e "${CYAN}Checking download of metafile for file ${files[$j,$j,$k,0]} in output file ${outputFiles[$i]}"
            fi
            if !(grep -q "DOWNLOADING metafile of ${files[$j,$j,$k,0]}2 from ${outputFiles[$j]:0:1}" "${outputFiles[$i]}")
            then
                warning=true
                if [[ $DEBUG = true ]]
                then
                    echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not receive metafile of ${files[$j,$j,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                fi
            fi

            if [[ $warning != true ]]
            then
                # Check each chunk
                for l in `seq 1 $maxNumberOfChunksPerFile`
                do
                    if [[ ${files[$j,$j,$k,$l]} != "" ]]
                    then
                        if [[ $DEBUG = true ]]
                        then
                            echo -e "${CYAN}Checking download of chunk $l for file ${files[$j,$j,$k,0]} in output file ${outputFiles[$i]}"
                        fi
                        if !(grep -q "DOWNLOADING ${files[$j,$j,$k,0]}2 chunk $l from ${outputFiles[$j]:0:1}" "${outputFiles[$i]}")
                        then
                            warning=true
                            if [[ $DEBUG = true ]]
                            then
                                echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not receive chunk $l of ${files[$j,$j,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                            fi
                        fi
                    fi
                done

                # Check the full reception of the file
                if [[ $warning != true ]] && !(grep -q "RECONSTRUCTED file ${files[$j,$j,$k,0]}2" "${outputFiles[$i]}")
                then
                    warning=true
                    if [[ $DEBUG = true ]]
                    then
                        echo -e "${WARNINGCOLOR}Node ${outputFiles[$i]:0:1} does not reconstruct file ${files[$j,$j,$k,0]} from ${outputFiles[$j]:0:1}${NC}"
                    fi
                fi
                if [[ $warning != true ]] && [[ $(diff "_Downloads/${files[$j,$j,$k,0]}2" "_SharedFiles/${files[$j,$j,$k,0]}") != "" ]]
                then
                    warning=true
                    if [[ $DEBUG = true ]]
                    then
                        echo -e "${WARNINGCOLOR}Reconstructed file ${files[$j,$j,$k,0]} does not match the original${NC}"
                    fi
                fi
            fi
        fi
        check_warning
    fi

    if [[ $TestSearch = true ]]
    then
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK search${NC}"
        fi
        # with a maximum budget of 32, we reach nodes 5 and n-4
        nodesSearched=""
        if [[ $numberOfPeers -gt 9 ]]
        then
            nodesSearched=("$(seq 0 5)" "$(seq $(($numberOfPeers-4)) $(($numberOfPeers-1)))")
        else
            nodesSearched="$(seq 0 $(($numberOfPeers-1)))"
        fi
        for search in $keywords
        do
            search="${search/,/}"
            search="${search/cc/c}"
            for i in $nodesSearched
            do
                if [[ $i -ne 0 ]]
                then
                    for j in `seq 0 $(($maxNumberOfFiles))`
                    do
                        if [[ ${files[$i,$i,$j,0]} =~ "$search" ]]
                        then
                            line="FOUND match ${files[$i,$i,$j,0]} at ${outputFiles[$i]:0:1} metafile=${files[$i,$i,$j,1]} chunks=1"
                            if !(grep -q "$line" "${outputFiles[0]}")
                            then
                                failed=true
                                if [[ "$DEBUG" == "true" ]] ; then
                                    echo -e "${RED}$line not present in ${outputFiles[0]}${NC}"
                                fi
                            fi
                        fi
                    done
                fi
            done
        done
        # here, the search is never finished
        if (grep -q "SEARCH FINISHED" "${outputFiles[$i]}")
        then
            failed=true
            if [[ "$DEBUG" == "true" ]] ; then
                echo -e "${RED}Unexpected end of search in ${outputFiles[0]}${NC}"
            fi
        fi
        check_failed
    fi

    if [[ $TestBlockchain = true ]]
    then
        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK found block${NC}"
        fi
        # each process will mine a block
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            if !(grep -Eq "FOUND-BLOCK 0{4}[a-f0-9]{60}" "${outputFiles[$i]}")
            then
                failed=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${RED}No block found in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
        check_failed

        if [[ $DEBUG = true ]]
        then
            echo -e "${BLUE}###CHECK chain update${NC}"
        fi
        # each process will add a block to the blockchain
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            # one block long chain
            if !(grep -Eq "CHAIN 0{4}[a-f0-9]{60}:0{64}:(image|local|global|neighbor|indirect|test|,)+" "${outputFiles[$i]}")
            then
                failed=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${RED}Chain initialization is not present in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
        # two blocks long chain: only mandatory for node A
        if !(grep -Eq "CHAIN 0{4}[a-f0-9]{60}:(0{4}[a-f0-9]{60}):(image|local|global|neighbor|indirect|test|,)+ \1:0{64}:(local|global|neighbor|indirect|test|,)+" "${outputFiles[0]}")
        then
            failed=true
            if [[ "$DEBUG" == "true" ]] ; then
                echo -e "${RED}No block added in ${outputFiles[0]}${NC}"
            fi
        fi
        # check consistancy of the hashes
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            if (cat "${outputFiles[$i]}" | awk '$0 ~ "\\[.*\\].*\\[.*\\]" { print $0; }' | sed 's/:\(.*\):[^\[]*\[\1//g' | grep -Eq '\[.*\] \[.*\]')
            then
                failed=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${RED}Inconsistent hashes in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
        # check unicity of files
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            if (grep -qE "[:,]([^],]+)( |,).*[:,]\1(,| )" "${outputFiles[$i]}")
            then
                failed=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${RED}File not unique in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
        check_failed
        # check that consensus has been reached (all files available everywhere
        for i in `seq 0 $(($numberOfPeers - 1))`
        do
            if !(grep "test" "${outputFiles[$i]}" | grep "local" | grep "global" | grep "neighbor" | grep "image" | grep -q "indirect")
            then
                warning=true
                if [[ "$DEBUG" == "true" ]] ; then
                    echo -e "${WARNINGCOLOR}Not all files present in ${outputFiles[$i]}${NC}"
                fi
            fi
        done
        check_warning
    fi
fi
