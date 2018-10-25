---
title: Swapperd

language_tabs:
  - shell

toc_footers:
  - <a href='https://github.com/republicprotocol/swapperd'>Checkout our GitHub</a>
  - <a href='https://republicprotocol.com'>Support by Republic Protocol</a>

includes:
  - errors

search: true
---

# Introduction

Welcome to Swapperd! You can use Swapperd execute cross-chain atomic swaps between Bitcoin, Ethereum, and ERC20 tokens.

# Installation

> Swapperd currently supports macOS, and Ubuntu. Run the following command in a terminal:

```shell
curl https://releases.republicprotocol.com/swapperd/install.sh -sSf | sh
```

Swapperd installs itself as a system service. In the event of an unexpected shutdown, Swapperd will automatically restart and resume all pending atomic swaps.

During installing, you will need to choose a `username` and `password`. These will be required when interacting with the authenticated HTTP endpoints exposed by Swapperd.

A `mneumonic` will be generated and printed to the terminal. Swapperd uses this `mneumonic`, with your `username` and `password`, to generate its Bitcoin and Ethereum private keys on-demand.

<aside class="success">
Swapperd nevers stores private keys to persistent storage. The `password` will be temporarily stored to persistent storage until the associated atomic swap has completed.
</aside>

<aside class="notice">
Backup the `username`, `paassowrd`, and `mnuemonic` generated during installation. Forgetting these could result in the loss of funds!
</aside>

# Authentication

> An example using HTTP Authentication:

```shell
curl -i     \
     -X GET \
     http://username:password@localhost:7777/balances
```

Swapperd protects itself using HTTP Basic Authentication. Swapperd protects all HTTP endpoints that require a Bitcoin or Ethereum private key.

<aside class="success">
Use the <code>username</code> and <code>password</code> that you entered during installation.
</aside>

# Swaps

## Executing an atomic swap

> Executing an atomic swap by initating first:

```shell
curl -i      \
     -X POST \
     -d '{ "sendToken": "BTC",                                          
           "receiveToken": "WBTC",                                      
           "sendAmount": "100000000",                                   
           "receiveAmount": "100000000",                                
           "sendTo": "mv1Pb8Ed7MA2wegbJQLZS3GzNHe9rpTBGK",              
           "receiveFrom": "0x5E6B16d705D81ec0822e6926E7841267Fa490b3E", 
           "shouldInitiateFirst": true }' \
     http://username:password@localhost:7777/swaps
```

> The response body is structured like this:

```json
{
  "id": "U1nWa7ggpDEFSh3ChZMxni6YZwy6SbcYpAy/Wc7CRRQ=",
  "sendToken": "BTC",
  "receiveToken": "WBTC",
  "sendAmount": "100000000",
  "receiveAmount": "100000000",
  "sendTo": "mzcJVzZgVcgmErPp2bu4DzXcdHpHA7wy1b",
  "receiveFrom": "0x43256f96601178Fd8594E02eed2e0d41f68DBb27",
  "timeLock": 1639947328,
  "secretHash": "4HOqdX0HpFtRz9bmPrYC2IYtAIOgsxCiN8L9+mp00zY=",
  "shouldInitiateFirst": true
}
```

### HTTP Request

`POST http://localhost:7777/swaps`

<aside class="success">
This is a protected HTTP endpoint.
</aside>

## Responding to an atomic swap

> Respond to an atomic swap by initating second:

```shell
curl -i      \
     -X POST \
     -d '{ "sendToken": "WBTC",
           "receiveToken": "BTC",
           "sendAmount": "100000000",
           "receiveAmount": "100000000",
           "sendTo": "0x5E6B16d705D81ec0822e6926E7841267Fa490b3E",
           "receiveFrom": "mv1Pb8Ed7MA2wegbJQLZS3GzNHe9rpTBGK",
           "timeLock": 1639947328,
           "secretHash": "4HOqdX0HpFtRz9bmPrYC2IYtAIOgsxCiN8L9+mp00zY=",
           "shouldInitiateFirst": false }' \
     http://username:password@localhost:7777/swaps
```

> The response body is structured like this:

```json
{
  "id": "S1Jn5MTLBqD8M2lm6vYjt1n2qy7XlW7sjHyIY3eInNA=",
  "sendToken": "WBTC",
  "receiveToken": "BTC",
  "sendAmount": "100000000",
  "receiveAmount": "100000000",
  "sendTo": "0x5E6B16d705D81ec0822e6926E7841267Fa490b3E",
  "receiveFrom": "mv1Pb8Ed7MA2wegbJQLZS3GzNHe9rpTBGK",
  "timeLock": 1639947328,
  "secretHash": "4HOqdX0HpFtRz9bmPrYC2IYtAIOgsxCiN8L9+mp00zY=",
  "shouldInitiateFirst": false
}
```

### HTTP Request

`POST http://localhost:7777/swaps`

<aside class="success">
This is a protected HTTP endpoint.
</aside>

## Get pending atomic swaps

```shell
curl -i     \
     -X GET \
     http://localhost:7777/swaps
```

> The response body is structured like this:

```json
{
  "swaps": [
    {
      "id": "S1Jn5MTLBqD8M2lm6vYjt1n2qy7XlW7sjHyIY3eInNA=",
      "status": 1
    }
  ]
}
```

### HTTP Request

`GET http://localhost:7777/swaps`

<aside class="notice">
This is not a protected HTTP endpoint.
</aside>

## Get balances

```shell
curl -i     \
     -X GET \
     http://username:password@localhost:7777/balances
```

> The response body is structured like this:

```json
{
  "balances": [
    {
      "token": "BTC",
      "address": "mv1Pb8Ed7MA2wegbJQLZS3GzNHe9rpTBGK",
      "amount": "13565"
    },
    {
      "token": "ETH",
      "address": "0x5E6B16d705D81ec0822e6926E7841267Fa490b3E",
      "amount": "990038930600000000"
    },
    {
      "token": "WBTC",
      "address": "0x5E6B16d705D81ec0822e6926E7841267Fa490b3E",
      "amount": "999846435"
    }
  ]
}
```

### HTTP Request

`GET http://localhost:7777/balances`

<aside class="success">
This is a protected HTTP endpoint.
</aside>

## Withdraw

```shell
curl -i      \
     -X POST \
     -d '{ "to": "mv1Pb8Ed7MA2wegbJQLZS3GzNHe9rpTBGK",
           "token": "BTC",
           "amount": "100000000" }' \
     http://username:password@localhost:7777/withdrawals
```

> The response body is structured like this:

```json
{}
```

### HTTP Request

`GET http://localhost:7777/withdrawals`

<aside class="success">
This is a protected HTTP endpoint.
</aside>