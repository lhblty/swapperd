package swapper

import (
	"encoding/base64"
	"fmt"

	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/swapperd/core/swapper/delayed"
	"github.com/republicprotocol/swapperd/core/swapper/immediate"
	"github.com/republicprotocol/swapperd/core/swapper/status"
	"github.com/republicprotocol/swapperd/foundation/blockchain"
	"github.com/republicprotocol/swapperd/foundation/swap"
	"github.com/republicprotocol/tau"
	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	LoadCosts(id swap.SwapID) (blockchain.Cost, blockchain.Cost)
	Receipts() ([]swap.SwapReceipt, error)
	PutReceipt(receipt swap.SwapReceipt) error

	PutSwap(blob swap.SwapBlob) error
	DeletePendingSwap(swap.SwapID) error
	PendingSwaps() ([]swap.SwapBlob, error)
	UpdateReceipt(receiptUpdate swap.ReceiptUpdate) error
}

type swapper struct {
	delayedSwapper   tau.Task
	immediateSwapper tau.Task
	status           tau.Task
	storage          Storage
}

func New(cap int, storage Storage, callback delayed.DelayCallback, builder immediate.ContractBuilder) tau.Task {
	delayedSwapperTask := delayed.New(cap, callback)
	immediateSwapperTask := immediate.New(cap, builder)
	swapStatusTask := status.New(cap)
	return tau.New(tau.NewIO(cap), &swapper{delayedSwapperTask, immediateSwapperTask, swapStatusTask, storage}, delayedSwapperTask, immediateSwapperTask, swapStatusTask)
}

func NewSwapper(delayedSwapperTask, immediateSwapperTask, statusTask tau.Task, storage Storage) tau.Reducer {
	return &swapper{delayedSwapperTask, immediateSwapperTask, statusTask, storage}
}

func (swapper *swapper) Reduce(msg tau.Message) tau.Message {
	switch msg := msg.(type) {
	case Bootload:
		return swapper.handleBootload(msg)
	case SwapRequest:
		return swapper.handleSwapRequest(msg)
	case immediate.ReceiptUpdate:
		return swapper.handleReceiptUpdate(swap.ReceiptUpdate(msg))
	case immediate.DeleteSwap:
		return swapper.handleDeleteSwap(msg.ID)
	case delayed.SwapRequest:
		return swapper.handleSwapRequest(SwapRequest(msg))
	case delayed.ReceiptUpdate:
		return swapper.handleReceiptUpdate(swap.ReceiptUpdate(msg))
	case delayed.DeleteSwap:
		return swapper.handleDeleteSwap(msg.ID)
	case status.ReceiptQuery:
		return swapper.handleReceiptQuery(msg)
	case tau.Error:
		return msg
	case tau.Tick:
		return swapper.handleTick(msg)
	default:
		return tau.NewError(fmt.Errorf("invalid message type in swapper: %T", msg))
	}
}

func (swapper *swapper) handleReceiptQuery(msg tau.Message) tau.Message {
	swapper.status.Send(msg)
	return nil
}

func (swapper *swapper) handleTick(msg tau.Message) tau.Message {
	swapper.status.Send(msg)
	swapper.immediateSwapper.Send(msg)
	swapper.delayedSwapper.Send(msg)
	return nil
}

func (swapper *swapper) handleReceiptUpdate(update swap.ReceiptUpdate) tau.Message {
	swapper.status.Send(status.ReceiptUpdate(update))
	if err := swapper.storage.UpdateReceipt(swap.ReceiptUpdate(update)); err != nil {
		return tau.NewError(err)
	}
	return nil
}

func (swapper *swapper) handleSwapRequest(msg SwapRequest) tau.Message {
	if err := swapper.storage.PutSwap(swap.SwapBlob(msg)); err != nil {
		return tau.NewError(err)
	}

	receipt := swap.NewSwapReceipt(swap.SwapBlob(msg))
	swapper.status.Send(status.Receipt(receipt))
	if err := swapper.storage.PutReceipt(receipt); err != nil {
		return tau.NewError(err)
	}

	if msg.Delay {
		swapper.delayedSwapper.Send(delayed.DelayedSwapRequest(msg))
		return nil
	}

	sendCost, receiveCost := swapper.storage.LoadCosts(msg.ID)
	swapper.immediateSwapper.Send(immediate.NewSwapRequest(swap.SwapBlob(msg), sendCost, receiveCost))
	return nil
}

func (swapper *swapper) handleBootload(msg Bootload) tau.Message {
	return tau.NewMessageBatch([]tau.Message{swapper.handleSwapperBootload(msg), swapper.handleStatusBootload(msg)})
}

func (swapper *swapper) handleStatusBootload(msg Bootload) tau.Message {
	// Loading historical swap receipts
	historicalReceipts, err := swapper.storage.Receipts()
	if err != nil {
		return tau.NewError(err)
	}

	co.ParForAll(historicalReceipts, func(i int) {
		swapper.status.Send(status.Receipt(historicalReceipts[i]))
	})

	return nil
}

func (swapper *swapper) handleSwapperBootload(msg Bootload) tau.Message {
	pendingSwaps, err := swapper.storage.PendingSwaps()
	if err != nil {
		return tau.NewError(err)
	}

	for _, pendingSwap := range pendingSwaps {
		hash, err := base64.StdEncoding.DecodeString(pendingSwap.PasswordHash)
		if pendingSwap.PasswordHash != "" && err != nil {
			continue
		}

		if pendingSwap.PasswordHash != "" && bcrypt.CompareHashAndPassword(hash, []byte(msg.Password)) != nil {
			continue
		}

		swapper.status.Send(status.ReceiptUpdate(swap.NewReceiptUpdate(pendingSwap.ID, func(receipt *swap.SwapReceipt) {
			receipt.Active = true
		})))

		pendingSwap.Password = msg.Password
		if pendingSwap.Delay {
			swapper.delayedSwapper.Send(delayed.DelayedSwapRequest(pendingSwap))
			continue
		}

		sendCost, receiveCost := swapper.storage.LoadCosts(pendingSwap.ID)
		swapper.immediateSwapper.Send(immediate.NewSwapRequest(pendingSwap, sendCost, receiveCost))
	}

	return nil
}

func (swapper *swapper) handleDeleteSwap(id swap.SwapID) tau.Message {
	if err := swapper.storage.DeletePendingSwap(id); err != nil {
		return tau.NewError(err)
	}
	return nil
}

type SwapRequest swap.SwapBlob

func (msg SwapRequest) IsMessage() {
}

type Bootload struct {
	Password string
}

func (msg Bootload) IsMessage() {
}
