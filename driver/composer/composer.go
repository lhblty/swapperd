package composer

import (
	"time"

	"github.com/republicprotocol/swapperd/adapter/binder"
	"github.com/republicprotocol/swapperd/adapter/callback"
	"github.com/republicprotocol/swapperd/adapter/db"
	"github.com/republicprotocol/swapperd/adapter/server"
	"github.com/republicprotocol/swapperd/driver/keystore"
	"github.com/republicprotocol/swapperd/driver/leveldb"
	"github.com/republicprotocol/swapperd/driver/logger"
	"github.com/republicprotocol/tau"
)

const BufferCapacity = 128

type composer struct {
	homeDir string
	network string
	port    string
}

type Composer interface {
	Run(doneCh <-chan struct{})
}

func New(homeDir, network, port string) Composer {
	return &composer{homeDir, network, port}
}

func (composer *composer) Run(done <-chan struct{}) {
	blockchain, err := keystore.Wallet(composer.homeDir, composer.network)
	if err != nil {
		panic(err)
	}

	ldb, err := leveldb.NewStore(composer.homeDir, composer.network)
	if err != nil {
		panic(err)
	}

	storage := db.New(ldb)
	logger := logger.NewStdOut()

	serverIO := tau.NewIO(BufferCapacity)
	go sendTicks(serverIO, done)
	go server.New(
		BufferCapacity,
		serverIO,
		storage,
		binder.NewBuilder(blockchain, logger),
		callback.New(),
		blockchain,
		logger,
	).Run(done)
}

func sendTicks(io tau.IO, done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			io.InputWriter() <- tau.NewTick(time.Now())
		}
	}
}
