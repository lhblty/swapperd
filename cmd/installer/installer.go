package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/republicprotocol/swapperd/driver/config"
	"github.com/republicprotocol/swapperd/driver/keystore"
	"github.com/republicprotocol/swapperd/utils"
)

func main() {
	loc := flag.String("location", utils.GetDefaultSwapperHome(), "Location of the swapper's home directory")
	repNet := flag.String("network", "mainnet", "Which republic protocol network to use")
	passphrase := flag.String("keyphrase", "", "Keyphrase to encrypt your key files")
	flag.Parse()

	if err := utils.CreateDir(*loc); err != nil {
		panic(err)
	}

	cfg, err := config.New(*loc, *repNet)
	if err != nil {
		panic(err)
	}
	if err := keystore.GenerateFile(cfg, *passphrase); err != nil {
		if err != keystore.NewErrKeyFileExists(cfg.HomeDir) {
			panic(err)
		}
	}
	addr := readAddress()
	cfg.AuthorizedAddresses = []string{addr}
	config.SaveToFile(fmt.Sprintf("%s/config-%s.json", *loc, *repNet), cfg)
}

func readAddress() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your RenEx Ethereum address: ")
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	addr := strings.TrimSpace(text)
	if len(addr) == 42 && addr[:2] == "0x" {
		addr = addr[2:]
	}
	addrBytes, err := hex.DecodeString(addr)
	if err != nil || len(addrBytes) != 20 {
		fmt.Println("Please enter a valid Ethereum address")
		return readAddress()
	}
	return "0x" + addr
}
