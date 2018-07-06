package watch

import (
	"github.com/republicprotocol/atom-go/services/swap"
)

type watch struct {
	network swap.Network
	info    swap.Info
	reqAtom swap.Atom
	resAtom swap.Atom
	wallet  Wallet
	str     swap.SwapStore
}

type Watch interface {
	Run([32]byte) error
	Status([32]byte) (string, error)
}

func NewWatch(network swap.Network, info swap.Info, wallet Wallet, reqAtom swap.Atom, resAtom swap.Atom, str swap.SwapStore) Watch {
	return &watch{
		network: network,
		info:    info,
		wallet:  wallet,
		reqAtom: reqAtom,
		resAtom: resAtom,
		str:     str,
	}
}

// Run runs the watch object on the given order id
func (watch *watch) Run(orderID [32]byte) error {
	match, err := watch.wallet.GetMatch(orderID)
	if err != nil {
		return err
	}

	if watch.reqAtom.PriorityCode() == match.RecieveCurrency() {
		addr, err := watch.reqAtom.GetKey().GetAddress()
		if err != nil {
			return err
		}
		watch.info.SetOwnerAddress(orderID, addr)
	} else {
		addr, err := watch.resAtom.GetKey().GetAddress()
		if err != nil {
			return err
		}
		watch.info.SetOwnerAddress(orderID, addr)
	}

	atomicSwap := swap.NewSwap(watch.reqAtom, watch.resAtom, watch.info, match, watch.network, watch.str)
	err = atomicSwap.Execute()
	return err
}

func (watch *watch) Status(orderID [32]byte) (string, error) {
	return watch.str.ReadStatus(orderID)
}
