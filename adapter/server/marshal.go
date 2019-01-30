package server

import (
	"encoding/json"

	"github.com/republicprotocol/swapperd/core/wallet/transfer"
	"github.com/republicprotocol/swapperd/foundation/blockchain"
	"github.com/republicprotocol/swapperd/foundation/swap"
)

type GetInfoResponse struct {
	Version              string                  `json:"version"`
	Bootloaded           bool                    `json:"bootloaded"`
	SupportedBlockchains []blockchain.Blockchain `json:"supportedBlockchains"`
	SupportedTokens      []blockchain.Token      `json:"supportedTokens"`
}

type GetSwapsResponse struct {
	Swaps []swap.SwapReceipt `json:"swaps"`
}

type GetSwapResponse swap.SwapReceipt

type GetBalanceResponse blockchain.Balance
type GetBalancesResponse map[blockchain.TokenName]blockchain.Balance

type GetAddressesResponse map[blockchain.TokenName]string
type GetAddressResponse string

// PostSwapsRequest is the object expected by a POST \swaps call
type PostSwapsRequest swap.SwapBlob

// PostSwapsResponse is the object returned on a successful POST \swaps call
type PostSwapsResponse struct {
	ID        swap.SwapID   `json:"id"`
	Swap      swap.SwapBlob `json:"swap,omitempty"`
	Signature string        `json:"signature,omitempty"`
}

// MarshalInitiatorPostSwapsResponse marshals an initiator's PostSwapsResponse
func MarshalInitiatorPostSwapsResponse(id swap.SwapID, swap swap.SwapBlob, signature string) PostSwapsResponse {
	return PostSwapsResponse{id, swap, signature}
}

// MarshalResponderPostSwapsResponse marshals an responder's PostSwapsResponse
func MarshalResponderPostSwapsResponse(id swap.SwapID) PostSwapsResponse {
	return PostSwapsResponse{ID: id}
}

type PostTransfersRequest struct {
	Token    string `json:"token"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Password string `json:"password"`
}

type PostTransfersResponse transfer.TransferReceipt

// GetSignatureResponseJSON is the object returned when the user calls GET
// /signature?type=json
type GetSignatureResponseJSON struct {
	Message   json.RawMessage `json:"message"`
	Signature string          `json:"signature"`
}

// MarshalGetSignatureResponseJSON marshals a message and signature to
// GetSignatureResponseJSON
func MarshalGetSignatureResponseJSON(message json.RawMessage, signature string) GetSignatureResponseJSON {
	return GetSignatureResponseJSON{message, signature}
}

// GetSignatureResponseString is the object returned when the user calls GET
// /signature?type=base64|hex
type GetSignatureResponseString struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// MarshalGetSignatureResponseString marshals a message and signature to
// GetSignatureResponseString
func MarshalGetSignatureResponseString(message, signature string) GetSignatureResponseString {
	return GetSignatureResponseString{message, signature}
}

// GetTransfersResponse is the object returned when the user calls GET
// /transfers
type GetTransfersResponse struct {
	Transfers []transfer.TransferReceipt `json:"transfers"`
}

// MarshalGetTransfersResponse marshals internal representation of transfer
// receipts to GetTransfersResponse
func MarshalGetTransfersResponse(receiptMap transfer.TransferReceiptMap) GetTransfersResponse {
	transfers := []transfer.TransferReceipt{}
	for _, receipt := range receiptMap {
		receipt.PasswordHash = ""
		transfers = append(transfers, receipt)
	}
	return GetTransfersResponse{
		Transfers: transfers,
	}
}
