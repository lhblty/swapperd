package swap

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"math/big"

	"github.com/republicprotocol/swapperd/foundation/blockchain"
)

// A SwapID uniquely identifies a Swap that is being executed.
type SwapID string

func RandomID() SwapID {
	id := [32]byte{}
	rand.Read(id[:])
	return SwapID(base64.StdEncoding.EncodeToString(id[:]))
}

const ExpiryUnit = int64(2 * 60 * 60)

// The SwapReceipt contains the swap details and the status.
type SwapReceipt struct {
	ID            SwapID          `json:"id"`
	SendToken     string          `json:"sendToken"`
	ReceiveToken  string          `json:"receiveToken"`
	SendAmount    string          `json:"sendAmount"`
	ReceiveAmount string          `json:"receiveAmount"`
	SendCost      blockchain.Cost `json:"sendCost"`
	ReceiveCost   blockchain.Cost `json:"receiveCost"`
	Timestamp     int64           `json:"timestamp"`
	Status        int             `json:"status"`
	Delay         bool            `json:"delay"`
	DelayInfo     json.RawMessage `json:"delayInfo,omitempty"`
}

// A Swap stores all of the information required to execute an atomic swap.
type Swap struct {
	ID              SwapID
	Token           blockchain.Token
	Value           *big.Int
	Fee             *big.Int
	BrokerFee       *big.Int
	SecretHash      [32]byte
	TimeLock        int64
	SpendingAddress string
	FundingAddress  string
	BrokerAddress   string
}

// A SwapBlob is used to encode a Swap for storage and transmission.
type SwapBlob struct {
	ID           SwapID `json:"id"`
	SendToken    string `json:"sendToken"`
	ReceiveToken string `json:"receiveToken"`

	// SendAmount and ReceiveAmount are decimal strings.
	SendFee              string `json:"sendFee"`
	SendAmount           string `json:"sendAmount"`
	ReceiveFee           string `json:"receiveFee"`
	ReceiveAmount        string `json:"receiveAmount"`
	MinimumReceiveAmount string `json:"minimumReceiveAmount,omitempty"`

	SendTo              string `json:"sendTo"`
	ReceiveFrom         string `json:"receiveFrom"`
	TimeLock            int64  `json:"timeLock"`
	SecretHash          string `json:"secretHash"`
	ShouldInitiateFirst bool   `json:"shouldInitiateFirst"`

	Delay            bool            `json:"delay,omitempty"`
	DelayInfo        json.RawMessage `json:"delayInfo,omitempty"`
	DelayCallbackURL string          `json:"delayCallbackUrl,omitempty"`

	BrokerFee              int64  `json:"brokerFee"` // in BIPs or (1/10000)
	BrokerSendTokenAddr    string `json:"brokerSendTokenAddr"`
	BrokerReceiveTokenAddr string `json:"brokerReceiveTokenAddr"`

	Password string `json:"password,omitempty"`
}

type ReceiptUpdate struct {
	ID     SwapID
	Update func(receipt *SwapReceipt)
}

func NewReceiptUpdate(id SwapID, update func(receipt *SwapReceipt)) ReceiptUpdate {
	return ReceiptUpdate{id, update}
}
