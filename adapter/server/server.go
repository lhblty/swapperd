package server

import (
	"github.com/republicprotocol/swapperd/adapter/wallet"
	"github.com/republicprotocol/swapperd/core/swapper"
	"github.com/republicprotocol/swapperd/core/swapper/delayed"
	"github.com/republicprotocol/swapperd/core/swapper/immediate"
	"github.com/republicprotocol/swapperd/core/transfer"
	"github.com/republicprotocol/tau"
	"github.com/sirupsen/logrus"
)

type server struct {
	swapperTask tau.Task
	walletTask  tau.Task

	logger logrus.FieldLogger
}

func New(cap int, io tau.IO, storage Storage, builder immediate.ContractBuilder, callback delayed.DelayCallback, wallet wallet.Wallet, logger logrus.FieldLogger) tau.Task {
	swapperTask := swapper.New(cap, storage, callback, builder)
	walletTask := transfer.New(cap, wallet, storage)
	return tau.New(io, &server{swapperTask, walletTask, logger}, swapperTask, walletTask)
}

func (server *server) Reduce(msg tau.Message) tau.Message {
	switch msg := msg.(type) {
	case tau.Tick:
		server.swapperTask.Send(msg)
		server.walletTask.Send(msg)
		return nil
	case tau.Error:
		server.logger.Error(msg)
		return nil
	default:
		server.logger.Errorf("unknown message type in server: %T", msg)
		return nil
	}
}
