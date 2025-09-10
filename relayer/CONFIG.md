[//]: # (Documentation generated from docs.toml - DO NOT EDIT.)
This document describes the TOML format for configuration.
## Example

```toml
ChainID = '<id>'

[[Tron.Nodes]]
Name = 'primary'
URL = '<full node http url>'
SolidityURL = '<solidity http url>'

```

## Global
```toml
ChainID = 'foobar' # Example
Enabled = true # Default
BalancePollPeriod = '5s' # Default
BroadcastChanSize = 4096 # Default
ConfirmPollPeriod = '500ms' # Default
OCR2CachePollPeriod = '5s' # Default
OCR2CacheTTL = '1m' # Default
RetentionPeriod = 0 # Default
ReapInterval = '1m' # Default
```


### ChainID
```toml
ChainID = 'foobar' # Example
```
ChainID is the Tron chain ID.

### Enabled
```toml
Enabled = true # Default
```
Enabled enables this chain.

### BalancePollPeriod
```toml
BalancePollPeriod = '5s' # Default
```
BalancePollPeriod is the poll period for balance monitoring

### BroadcastChanSize
```toml
BroadcastChanSize = 4096 # Default
```
BroadcastChanSize is the transaction broadcast channel size

### ConfirmPollPeriod
```toml
ConfirmPollPeriod = '500ms' # Default
```
ConfirmPollPeriod is the polling period for transaction confirmation

### OCR2CachePollPeriod
```toml
OCR2CachePollPeriod = '5s' # Default
```
OCR2CachePollPeriod is the polling period for OCR2 contract cache

### OCR2CacheTTL
```toml
OCR2CacheTTL = '1m' # Default
```
OCR2CacheTTL is the time to live for OCR2 contract cache

### RetentionPeriod
```toml
RetentionPeriod = 0 # Default
```
RetentionPeriod is the time for the tx manager to retain txes.

### ReapInterval
```toml
ReapInterval = '1m' # Default
```
ReapInterval is how often the tx manager cleans up old txes.

## Nodes
```toml
[[Nodes]]
Name = 'primary' # Example
URL = 'https://api.trongrid.io/wallet' # Example
SolidityURL = 'http://api.trongrid.io/wallet' # Example
```


### Name
```toml
Name = 'primary' # Example
```
Name is a unique (per-chain) identifier for this node.

### URL
```toml
URL = 'https://api.trongrid.io/wallet' # Example
```
URL is the full node HTTP endpoint for this node.

### SolidityURL
```toml
SolidityURL = 'http://api.trongrid.io/wallet' # Example
```
SolidityURL is the solidity node HTTP endpoint for this node.

