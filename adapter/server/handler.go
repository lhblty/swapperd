package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	"github.com/republicprotocol/swapperd/adapter/wallet"
	coreWallet "github.com/republicprotocol/swapperd/core/wallet"
	"github.com/republicprotocol/swapperd/core/wallet/swapper"
	"github.com/republicprotocol/swapperd/core/wallet/swapper/status"
	"github.com/republicprotocol/swapperd/core/wallet/transfer"
	"github.com/republicprotocol/swapperd/foundation/blockchain"
	"github.com/republicprotocol/swapperd/foundation/swap"
	"github.com/republicprotocol/tau"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
)

var ErrHandlerIsShuttingDown = fmt.Errorf("Http handler is shutting down")

type handler struct {
	bootloaded map[string]bool
	wallet     wallet.Wallet
	buffer     chan tau.Message
	done       chan struct{}
}

// The Handler for swapperd requests
type Handler interface {
	GetID(password string, idType string) (string, error)
	GetInfo(password string) GetInfoResponse
	GetSwap(password string, id swap.SwapID) (GetSwapResponse, error)
	GetSwaps(password string) (GetSwapsResponse, error)
	GetBalances(password string) (GetBalancesResponse, error)
	GetBalance(password string, token blockchain.Token) (GetBalanceResponse, error)
	GetAddresses(password string) (GetAddressesResponse, error)
	GetAddress(password string, token blockchain.Token) (GetAddressResponse, error)
	GetTransfers(password string) (GetTransfersResponse, error)
	GetJSONSignature(password string, message json.RawMessage) (GetSignatureResponseJSON, error)
	GetBase64Signature(password string, message string) (GetSignatureResponseString, error)
	GetHexSignature(password string, message string) (GetSignatureResponseString, error)
	PostTransfers(PostTransfersRequest) (PostTransfersResponse, error)
	PostSwaps(PostSwapsRequest) (PostSwapsResponse, error)
	PostDelayedSwaps(PostSwapsRequest) error

	Receive() (tau.Message, error)
	Write(msg tau.Message)
	ShutDown()
}

func NewHandler(cap int, wallet wallet.Wallet) Handler {
	return &handler{
		bootloaded: map[string]bool{},
		wallet:     wallet,
		buffer:     make(chan tau.Message, cap),
		done:       make(chan struct{}),
	}
}

func (handler *handler) GetInfo(password string) GetInfoResponse {
	handler.bootload(password)
	return GetInfoResponse{
		Version:         "0.3.0",
		Bootloaded:      handler.bootloaded[passwordHash(password)],
		SupportedTokens: handler.wallet.SupportedTokens(),
	}
}

func (handler *handler) GetAddresses(password string) (GetAddressesResponse, error) {
	handler.bootload(password)
	return handler.wallet.Addresses(password)
}

func (handler *handler) GetAddress(password string, token blockchain.Token) (GetAddressResponse, error) {
	handler.bootload(password)
	address, err := handler.wallet.GetAddress(password, token.Blockchain)
	return GetAddressResponse(address), err
}

func (handler *handler) GetSwaps(password string) (GetSwapsResponse, error) {
	handler.bootload(password)
	resp := GetSwapsResponse{}
	swapReceipts := handler.getSwapReceipts(password)
	for _, receipt := range swapReceipts {
		passwordHash, err := base64.StdEncoding.DecodeString(receipt.PasswordHash)
		if receipt.PasswordHash != "" && err != nil {
			return resp, fmt.Errorf("corrupted password")
		}

		if receipt.PasswordHash != "" && bcrypt.CompareHashAndPassword(passwordHash, []byte(password)) != nil {
			continue
		}
		receipt.PasswordHash = ""
		resp.Swaps = append(resp.Swaps, receipt)
	}
	return resp, nil
}

func (handler *handler) GetSwap(password string, id swap.SwapID) (GetSwapResponse, error) {
	handler.bootload(password)
	swapReceipts := handler.getSwapReceipts(password)
	receipt, ok := swapReceipts[id]
	if !ok {
		return GetSwapResponse{}, fmt.Errorf("swap receipt not found")
	}
	return GetSwapResponse(receipt), nil
}

func (handler *handler) getSwapReceipts(password string) map[swap.SwapID]swap.SwapReceipt {
	handler.bootload(password)
	responder := make(chan map[swap.SwapID]swap.SwapReceipt)
	handler.Write(coreWallet.NewSwapperRequest(status.ReceiptQuery{Responder: responder}))
	swapReceipts := <-responder
	return swapReceipts
}

func (handler *handler) GetBalances(password string) (GetBalancesResponse, error) {
	handler.bootload(password)
	balanceMap, err := handler.wallet.Balances(password)
	return GetBalancesResponse(balanceMap), err
}

func (handler *handler) GetBalance(password string, token blockchain.Token) (GetBalanceResponse, error) {
	handler.bootload(password)
	balance, err := handler.wallet.Balance(password, token)
	return GetBalanceResponse(balance), err
}

func (handler *handler) GetTransfers(password string) (GetTransfersResponse, error) {
	handler.bootload(password)
	responder := make(chan transfer.TransferReceiptMap, 1)
	handler.Write(coreWallet.NewTransferRequest(transfer.TransferReceiptRequest{responder}))
	response := <-responder

	receiptMap := transfer.TransferReceiptMap{}
	for key, receipt := range response {
		passwordHash, err := base64.StdEncoding.DecodeString(receipt.PasswordHash)
		if receipt.PasswordHash != "" && err != nil {
			return GetTransfersResponse{}, err
		}

		if receipt.PasswordHash != "" && bcrypt.CompareHashAndPassword(passwordHash, []byte(password)) != nil {
			continue
		}

		receiptMap[key] = receipt
	}
	return MarshalGetTransfersResponse(receiptMap), nil
}

func (handler *handler) PostSwaps(swapReq PostSwapsRequest) (PostSwapsResponse, error) {
	handler.bootload(swapReq.Password)

	blob, err := handler.patchSwap(swap.SwapBlob(swapReq))
	if err != nil {
		return PostSwapsResponse{}, err
	}

	handler.Write(coreWallet.NewSwapperRequest(swapper.SwapRequest(blob)))
	return handler.buildSwapResponse(blob)
}

func (handler *handler) PostDelayedSwaps(swapReq PostSwapsRequest) error {
	handler.bootload(swapReq.Password)

	blob, err := handler.patchDelayedSwap(swap.SwapBlob(swapReq))
	if err != nil {
		return err
	}

	blob, err = handler.signDelayInfo(blob)
	if err != nil {
		return err
	}

	handler.Write(coreWallet.NewSwapperRequest(swapper.SwapRequest(blob)))
	return nil
}

func (handler *handler) PostTransfers(req PostTransfersRequest) (PostTransfersResponse, error) {
	handler.bootload(req.Password)
	response := PostTransfersResponse{}
	token, err := blockchain.PatchToken(req.Token)
	if err != nil {
		return response, err
	}

	responder := make(chan transfer.TransferReceipt, 1)
	if err := handler.wallet.VerifyAddress(token.Blockchain, req.To); err != nil {
		return response, err
	}

	amount, ok := big.NewInt(0).SetString(req.Amount, 10)
	if !ok {
		return response, fmt.Errorf("invalid amount %s", req.Amount)
	}

	if err := handler.wallet.VerifyBalance(req.Password, token, amount); err != nil {
		return response, err
	}

	txCost, err := token.TransactionCost(amount)
	if err != nil {
		return response, err
	}

	handler.Write(coreWallet.NewTransferRequest(transfer.NewTransferRequest(req.Password, token, req.To, amount, txCost, responder)))
	transferReceipt := <-responder
	return PostTransfersResponse(transferReceipt), nil
}

func (handler *handler) GetID(password, idType string) (string, error) {
	handler.bootload(password)
	id, err := handler.wallet.ID(password, idType)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (handler *handler) GetJSONSignature(password string, message json.RawMessage) (GetSignatureResponseJSON, error) {
	handler.bootload(password)
	sig, err := handler.sign(password, message)
	if err != nil {
		return GetSignatureResponseJSON{}, err
	}
	return GetSignatureResponseJSON{
		Message:   message,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}, nil
}

func (handler *handler) GetBase64Signature(password string, message string) (GetSignatureResponseString, error) {
	handler.bootload(password)
	msg, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return GetSignatureResponseString{}, err
	}

	sig, err := handler.sign(password, msg)
	if err != nil {
		return GetSignatureResponseString{}, err
	}

	return GetSignatureResponseString{
		Message:   message,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}, nil
}

func (handler *handler) GetHexSignature(password string, message string) (GetSignatureResponseString, error) {
	handler.bootload(password)
	if len(message) > 2 && message[:2] == "0x" {
		message = message[2:]
	}
	msg, err := hex.DecodeString(message)
	if err != nil {
		return GetSignatureResponseString{}, err
	}

	sig, err := handler.sign(password, msg)
	if err != nil {
		return GetSignatureResponseString{}, err
	}

	return GetSignatureResponseString{
		Message:   message,
		Signature: hex.EncodeToString(sig),
	}, nil
}

func (handler *handler) Receive() (tau.Message, error) {
	select {
	case <-handler.done:
		return nil, ErrHandlerIsShuttingDown
	case message := <-handler.buffer:
		return message, nil
	}
}

func (handler *handler) Write(msg tau.Message) {
	handler.buffer <- msg
}

func (handler *handler) ShutDown() {
	close(handler.done)
}

func (handler *handler) bootload(password string) {
	if !handler.bootloaded[passwordHash(password)] {
		handler.Write(coreWallet.NewSwapperRequest(swapper.Bootload{password}))
		handler.Write(coreWallet.NewTransferRequest(transfer.Bootload{}))
		handler.bootloaded[passwordHash(password)] = true
	}
}

func (handler *handler) patchSwap(swapBlob swap.SwapBlob) (swap.SwapBlob, error) {
	sendToken, err := blockchain.PatchToken(string(swapBlob.SendToken))
	if err != nil {
		return swapBlob, err
	}

	if err := handler.wallet.VerifyAddress(sendToken.Blockchain, swapBlob.SendTo); err != nil {
		return swapBlob, err
	}

	receiveToken, err := blockchain.PatchToken(string(swapBlob.ReceiveToken))
	if err != nil {
		return swapBlob, err
	}

	if err := handler.wallet.VerifyAddress(receiveToken.Blockchain, swapBlob.ReceiveFrom); err != nil {
		return swapBlob, err
	}

	if swapBlob.WithdrawAddress != "" {
		if err := handler.wallet.VerifyAddress(receiveToken.Blockchain, swapBlob.WithdrawAddress); err != nil {
			return swapBlob, err
		}
	}

	if err := handler.verifySendAmount(swapBlob.Password, sendToken, swapBlob.SendAmount); err != nil {
		return swapBlob, err
	}

	if err := handler.verifyReceiveAmount(swapBlob.Password, receiveToken); err != nil {
		return swapBlob, err
	}

	swapID := [32]byte{}
	rand.Read(swapID[:])
	swapBlob.ID = swap.SwapID(base64.StdEncoding.EncodeToString(swapID[:]))
	secret := [32]byte{}
	if swapBlob.ShouldInitiateFirst {
		swapBlob.TimeLock = time.Now().Unix() + 3*swap.ExpiryUnit
		secret = genereateSecret(swapBlob.Password, swapBlob.ID)
		hash := sha256.Sum256(secret[:])
		swapBlob.SecretHash = base64.StdEncoding.EncodeToString(hash[:])
		return swapBlob, nil
	}

	secretHash, err := base64.StdEncoding.DecodeString(swapBlob.SecretHash)
	if len(secretHash) != 32 || err != nil {
		return swapBlob, fmt.Errorf("invalid secret hash")
	}
	if time.Now().Unix()+2*swap.ExpiryUnit > swapBlob.TimeLock {
		return swapBlob, fmt.Errorf("not enough time to do the atomic swap")
	}
	return swapBlob, nil
}

func (handler *handler) patchDelayedSwap(blob swap.SwapBlob) (swap.SwapBlob, error) {
	if blob.DelayCallbackURL == "" {
		return blob, fmt.Errorf("delay url cannot be empty")
	}

	swapID := [32]byte{}
	rand.Read(swapID[:])
	blob.ID = swap.SwapID(base64.StdEncoding.EncodeToString(swapID[:]))

	sendToken, err := blockchain.PatchToken(string(blob.SendToken))
	if err != nil {
		return blob, err
	}
	if err := handler.verifySendAmount(blob.Password, sendToken, blob.SendAmount); err != nil {
		return blob, err
	}

	receiveToken, err := blockchain.PatchToken(string(blob.ReceiveToken))
	if err != nil {
		return blob, err
	}
	if err := handler.verifyReceiveAmount(blob.Password, receiveToken); err != nil {
		return blob, err
	}

	secret := genereateSecret(blob.Password, blob.ID)
	secretHash := sha256.Sum256(secret[:])
	blob.SecretHash = base64.StdEncoding.EncodeToString(secretHash[:])
	blob.TimeLock = time.Now().Unix() + 3*swap.ExpiryUnit
	return blob, nil
}

func (handler *handler) verifySendAmount(password string, token blockchain.Token, amount string) error {
	sendAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return fmt.Errorf("invalid send amount")
	}
	return handler.wallet.VerifyBalance(password, token, sendAmount)
}

func (handler *handler) verifyReceiveAmount(password string, token blockchain.Token) error {
	return handler.wallet.VerifyBalance(password, token, nil)
}

func (handler *handler) signDelayInfo(blob swap.SwapBlob) (swap.SwapBlob, error) {
	delayInfoSig, err := handler.sign(blob.Password, blob.DelayInfo)
	if err != nil {
		return blob, fmt.Errorf("failed to sign delay info: %v", err)
	}

	signedDelayInfo, err := json.Marshal(struct {
		Message   json.RawMessage `json:"message"`
		Signature string          `json:"signature"`
	}{
		Message:   blob.DelayInfo,
		Signature: base64.StdEncoding.EncodeToString(delayInfoSig),
	})
	if err != nil {
		return blob, fmt.Errorf("unable to marshal signed delay info: %v", err)
	}

	blob.DelayInfo = signedDelayInfo
	return blob, nil
}

func (handler *handler) buildSwapResponse(blob swap.SwapBlob) (PostSwapsResponse, error) {
	responseBlob := swap.SwapBlob{}
	responseBlob.SendToken = blob.ReceiveToken
	responseBlob.ReceiveToken = blob.SendToken
	responseBlob.SendAmount = blob.ReceiveAmount
	responseBlob.ReceiveAmount = blob.SendAmount
	swapResponse := PostSwapsResponse{}

	sendToken, err := blockchain.PatchToken(string(responseBlob.SendToken))
	if err != nil {
		return swapResponse, err
	}

	receiveToken, err := blockchain.PatchToken(string(responseBlob.ReceiveToken))
	if err != nil {
		return swapResponse, err
	}

	sendTo, err := handler.wallet.GetAddress(blob.Password, sendToken.Blockchain)
	if err != nil {
		return swapResponse, err
	}

	receiveFrom, err := handler.wallet.GetAddress(blob.Password, receiveToken.Blockchain)
	if err != nil {
		return swapResponse, err
	}

	responseBlob.SendTo = sendTo
	responseBlob.ReceiveFrom = receiveFrom
	responseBlob.SecretHash = blob.SecretHash
	responseBlob.TimeLock = blob.TimeLock

	responseBlob.BrokerFee = blob.BrokerFee
	responseBlob.BrokerSendTokenAddr = blob.BrokerReceiveTokenAddr
	responseBlob.BrokerReceiveTokenAddr = blob.BrokerSendTokenAddr

	responseBlobBytes, err := json.Marshal(responseBlob)
	if err != nil {
		return swapResponse, err
	}

	responseBlobSig, err := handler.sign(blob.Password, responseBlobBytes)
	if err != nil {
		return swapResponse, err
	}

	if blob.ShouldInitiateFirst {
		swapResponse.Swap = responseBlob
		swapResponse.Signature = base64.StdEncoding.EncodeToString(responseBlobSig)
	}

	if blob.ResponseURL != "" {
		data, err := json.MarshalIndent(swapResponse, "", "  ")
		if err != nil {
			return swapResponse, err
		}
		buf := bytes.NewBuffer(data)

		resp, err := http.Post(blob.ResponseURL, "application/json", buf)
		if err != nil {
			return swapResponse, err
		}

		if resp.StatusCode != 200 {
			respBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return swapResponse, err
			}

			return swapResponse, fmt.Errorf("unexpected status code (%d) while"+
				"posting to the response url: %s", resp.StatusCode, respBytes)
		}
	}
	swapResponse.ID = blob.ID
	return swapResponse, nil
}

func (handler *handler) sign(password string, message []byte) ([]byte, error) {
	signer, err := handler.wallet.ECDSASigner(password)
	if err != nil {
		return nil, fmt.Errorf("unable to load ecdsa signer: %v", err)
	}
	hash := sha3.Sum256(message)
	sig, err := signer.Sign(hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign swap response: %v", err)
	}
	return sig, nil
}

func genereateSecret(password string, id swap.SwapID) [32]byte {
	return sha3.Sum256(append([]byte(password), []byte(id)...))
}

func passwordHash(password string) string {
	passwordHash32 := sha3.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(passwordHash32[:])
}
