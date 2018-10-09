package http

import (
	"errors"
	"strconv"

	"github.com/republicprotocol/swapperd/adapter/btc"
	"github.com/republicprotocol/swapperd/adapter/config"
	"github.com/republicprotocol/swapperd/adapter/eth"
	"github.com/republicprotocol/swapperd/adapter/keystore"
	"github.com/republicprotocol/swapperd/domain/token"
	"github.com/republicprotocol/swapperd/service/renex"
)

var ErrInvalidSignatureLength = errors.New("invalid signature length")
var ErrInvalidOrderIDLength = errors.New("invalid order id length")

type Adapter interface {
	WhoAmI(challenge string) (WhoAmISigned, error)
	PostOrder(order PostOrderRequest) (PostOrderResponse, error)
	GetStatus(orderID string) (Status, error)
	GetBalances() (Balances, error)
}

type adapter struct {
	config config.Config
	keystr keystore.Keystore
	renex  renex.RenEx
}

func NewAdapter(config config.Config, keystr keystore.Keystore, renExSwapper renex.RenEx) Adapter {
	return &adapter{
		config: config,
		keystr: keystr,
		renex:  renExSwapper,
	}
}

func (adapter *adapter) WhoAmI(challenge string) (WhoAmISigned, error) {
	whoAmI := NewWhoAmI(challenge, adapter.config)
	infoBytes, err := MarshalWhoAmI(whoAmI)
	if err != nil {
		return WhoAmISigned{}, err
	}
	ethKey := adapter.keystr.GetKey(token.ETH).(keystore.EthereumKey)
	sig, err := ethKey.Sign(infoBytes)
	return WhoAmISigned{
		Signature: MarshalSignature(sig),
		WhoAmI:    whoAmI,
	}, nil
}

func (adapter *adapter) PostOrder(order PostOrderRequest) (PostOrderResponse, error) {
	orderID, err := UnmarshalOrderID(order.OrderID)
	if err != nil {
		return PostOrderResponse{}, err
	}

	if err := adapter.renex.Add(orderID); err != nil {
		return PostOrderResponse{}, err
	}

	key := adapter.keystr.GetKey(token.ETH).(keystore.EthereumKey)
	sig, err := key.Sign(orderID[:])
	return PostOrderResponse{
		order.OrderID,
		MarshalSignature(sig),
	}, nil
}

func (adapter *adapter) GetStatus(orderID string) (Status, error) {
	id, err := UnmarshalOrderID(orderID)
	if err != nil {
		return Status{}, err
	}
	status := adapter.renex.Status(id)
	return Status{
		OrderID: orderID,
		Status:  string(status),
	}, nil
}

func (adapter *adapter) GetBalances() (Balances, error) {
	ethBal, err := ethereumBalance(
		adapter.config,
		adapter.keystr.GetKey(token.ETH).(keystore.EthereumKey),
	)
	if err != nil {
		return Balances{}, err
	}
	btcBal := bitcoinBalance(
		adapter.config,
		adapter.keystr.GetKey(token.BTC).(keystore.BitcoinKey),
	)
	return Balances{
		Ethereum: ethBal,
		Bitcoin:  btcBal,
	}, nil
}

func bitcoinBalance(conf config.Config, key keystore.BitcoinKey) Balance {
	conn := btc.NewConnWithConfig(conf.Bitcoin)
	balance := conn.Balance(key.AddressString, 0)
	return Balance{
		Address: key.AddressString,
		Amount:  strconv.FormatInt(balance, 10),
	}
}

func ethereumBalance(conf config.Config, key keystore.EthereumKey) (Balance, error) {
	conn, err := eth.NewConnWithConfig(conf.Ethereum)
	if err != nil {
		return Balance{}, err
	}
	bal, err := conn.Balance(key.Address)
	if err != nil {
		return Balance{}, err
	}
	return Balance{
		Address: key.Address.String(),
		Amount:  bal.String(),
	}, nil
}
