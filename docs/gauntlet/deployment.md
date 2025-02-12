# Tron Feed Deployment with G++

## Setup

Clone the G++ repo:

```sh
git clone org-25111032@github.com:smartcontractkit/gauntlet-plus-plus.git
```

Navigate to the cloned directory and run `nix develop`. Once you enter the Nix shell, run the following commands:

```sh
yarn gauntlet plugins link ./packages-ethereum/extension-ethereum
yarn gauntlet plugins link ./packages-ethereum/operations-access-controller
yarn gauntlet plugins link ./packages-tron/extension-tron
yarn gauntlet plugins link ./packages-tron/operations-data-feeds
```

To confirm that the Tron plugins are linked correctly, you should be able to see a set of `tron/data-feeds/*` commands when running `yarn gauntlet ls`.

Once the Tron plugins are linked, create a JSON config file with the following contents:

```json
{
  "providers": [
    {
      "type": "@chainlink/gauntlet-tron/lib/tronweb",
      "name": "tronweb",
      "input": {
        "fullHost": "https://nile.trongrid.io/",
        "solidityNode": "https://nile.trongrid.io/"
      }
    },
    {
      "type": "@chainlink/gauntlet-tron/signer",
      "name": "raw-pk",
      "input": {
        "privateKey": "<PRIVATE_KEY>",
        "debug": true
      }
    },
    {
      "type": "@chainlink/gauntlet-tron/energy-estimator",
      "name": "basic-estimator",
      "input": {}
    }
  ],
  "datasources": [
    {
      "name": "erc20-abi",
      "input": {}
    }
  ]
}
```

For the `<PRIVATE_KEY>`, please reach out to Calvin Wang for a Tron Nile testnet wallet with testnet funds.

## Steps

**IMPORTANT**: when deploying contracts, the contract address returned at the very end of the `execute` stage when running a G++ operation is the real address of the deployed contract. Do not use the contract address shown at the end of the`plan` stage (before the transaction is broadcast) as G++ edits some fee parameters afterwards so the final contract address ends up being different.

**NOTE**: _Tron addresses can be represented in three formats - the Tron G++ plugin supports all three types_, so you can use whichever is most convenient. By default, the G++ deployment output will return the deployed contract address in Tron hex format. To convert between address formats, e.g. to look up an address in the block explorer or update RDD, use this online converter. Here are the address formats for reference:

- _Base58_ (e.g. TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv) - this is what is most supported across the Tron ecosystem, and you will have to use this format to interact with the Tron block explorer (<https://nile.tronscan.org/>)

- _Tron hex_ (e.g. 41f5c86a1c3400d0429350143322811bdd66d471fb) - this is the hex representation of Tron addresses, which often shows up in the Tron logs or api responses. They are 21 bytes long with the first byte always being 41 and no 0x prefix.

- _EVM hex_ (e.g. 0xf5c86A1C3400d0429350143322811BDd66D471Fb) - this the EVM-compatible format, which is essentially the same as the Tron hex format but is 20 bytes long without the 41 prefix, and instead has an 0x prefix. We use this format in RDD due to compatibility with downstream RDD consumers (e.g. Atlas) which expect an EVM address format.

### Deploy an Access Controller

For staging, you only need to deploy one access controller and the address can be reused. Access controllers can be configured post-deployment as required (for prod testnet?), but that isn't covered in this guide.

#### Input

```json
{}
```

#### Command

```sh
yarn gauntlet execute \
  -o tron/data-feeds/access-controller:deploy-write \
  -c config.json
```

#### RDD

- Note the address of the access controller, it will be needed later for RDD

### Deploy an Aggregator

Every feed is associated with one aggregator contract.

#### Input

- The value of the `link` field (e.g. 4100...) is actually the zero address (the 41 prefix can be replaced by 0x) - at the moment, on-chain billing is unused

- The min and max answer can usually stay the same for all feeds

- The `decimals` may vary depending on the feed

```json
{
  "description": "BTC / USD - Staging",
  "decimals": "18",
  "minAnswer": "1",
  "maxAnswer": "95780971304118053647396689196894323976171195136475135",
  "link": "410000000000000000000000000000000000000000",
  "billingAccessController": "<ACCESS_CONTROLLER_ADDRESS>",
  "requesterAccessController": "<ACCESS_CONTROLLER_ADDRESS>"
}
```

#### Command

```sh
yarn gauntlet execute \
  -o tron/data-feeds/aggregator:deploy \
  -c config.json \
  -i input.json
```

#### RDD

- Note the address of the aggregator, it will be needed later for RDD

### Deploy an Aggregator Proxy

#### Input

```json
{
  "aggregator": "<AGREGATOR_ADDRESS>",
  "accessController": "<ACCESS_CONTROLLER_ADDRESS>"
}
```

#### Command

```sh
yarn gauntlet execute \
  -o tron/data-feeds/aggregator-proxy:deploy \
  -c config.json \
  -i input.json
```

#### RDD

- Note the address of the aggregator proxy, it will be needed later for RDD

### Set the OCR Config

#### Input

- Make sure `alphaReportPpb` and `alphaAcceptPpb` are set to 3000000 for stablecoins
- The `configEncodingSecret` and `signerSecret` should be set accordingly for prod testnet and prod mainnet

##### For Staging

```json
{
  "address": "<AGGREGATOR_ADDRESS>",
  "config": {
    "f": 1,
    "transmitters": [
      "TTuSB24ukWJS3Xken6cPcHC5wkYwyhdHJ5",
      "TQykTPZnRcVCSAG1mcN7vyNtg8f5n7t9dL",
      "THiDRb4tDiCKAGcZvungG8Yni3ExmRusn9",
      "TCFSe4iDg1q5zzyf8exwGkgVK69hsqtJJR"
    ],
    "signers": [
      "41217e693404e6bce9195229c856bbc3af0143b351",
      "410d98ed0d94541b853b79fd5e50d5b4992ef9e2df",
      "414196f2e38da17693fbf41fb89167ede64f490d98",
      "4112dec6f513a6572e2ff1c1bdcf94dd3ada8f1c17"
    ],
    "offchainConfigVersion": 2,
    "offchainConfig": {
      "deltaProgress": "23s",
      "deltaResend": "12s",
      "deltaRound": "10s",
      "deltaGrace": "1s",
      "deltaStage": "20s",
      "rMax": 6,
      "s": [1, 1, 1, 1],
      "peerIds": [
        "12D3KooWF2Mpk5E4Tv8xPSi2PJtgjbBSDnugo9cydiShNnhpsfpa",
        "12D3KooWDxqq1HXjfgXsc5btEfBgemAoQnanFdjGkHahQsZmat79",
        "12D3KooWHiGF8n6W4TkNRBvwb2m5UFaZuwUWiVPiTGM45c41BRZD",
        "12D3KooWEhGoxcqwWXvimokaoujojthbz7HNsVHiuQyxzERKHBtr"
      ],
      "maxDurationQuery": "500ms",
      "maxDurationObservation": "5s",
      "maxDurationReport": "10s",
      "maxDurationShouldAcceptFinalizedReport": "10s",
      "maxDurationShouldTransmitAcceptedReport": "10s",
      "reportingPluginConfig": {
        "alphaReportInfinite": false,
        "alphaReportPpb": "5000000",
        "alphaAcceptInfinite": false,
        "alphaAcceptPpb": "5000000",
        "deltaC": "24h"
      },
      "configEncodingSecret": "test secret",
      "signerSecret": "test secret",
      "offchainPublicKeys": [
        "ocr2off_tron_3c95e3f4701946cab5bd68bc6245ffdde2c67ff57808fe12b998f75626bebbf0",
        "ocr2off_tron_03452b02f7f40ba766733735ecae22033b969eb8ed5600c429d948742c8b78ed",
        "ocr2off_tron_a1706997565cf9b79680c3e4639f3824a8c4c2641faa22bd8059629e4c2fe8b5",
        "ocr2off_tron_52738a894649ada1984c85180cd6efaab66f99c96c6d4e5fd684a001bf6b65a4"
      ],
      "configPublicKeys": [
        "ocr2cfg_tron_a939d201a89ebe0c7797af0beb17ec6e2699fd2fb829fea6225abd0da3af2400",
        "ocr2cfg_tron_f0e8ac07c95614c9cac53ba67cff4352102c72af4afdb2fda01b88f607ff405f",
        "ocr2cfg_tron_65dbf78a76b0517cdb6543e1765ebd80b27ea64d5f8d82934cf1c0377a55937a",
        "ocr2cfg_tron_07153ac15f253b52eb975424014e6a501802654e16fe28a801ef3c978000650b"
      ]
    }
  }
}
```

##### For Prod Testnet

```json
{
  "address": "<AGGREGATOR_ADDRESS>",
  "config": {
    "f": 1,
    "transmitters": [
      "TMYcfmbHzVQ853XJFqoPir8iQj5dWsCwbX",
      "TUiQyM8U6GULPMmQUGEsNv8A8LdpJmDupd",
      "TSxRLZTbE287fvaJJTye6iuW6CXj1CxZEh",
      "TK9BTTkugRBdiDJpiRBs4cMTeVGDCKRkFY"
    ],
    "signers": [
      "414fc98f0a1410e03314a67b05ac7e3997279088a3",
      "419bea76aaf7ad28f36bebfbf3fde9341dd051a289",
      "4177668ae43e1c766423eb4e2c53305b5c3ca3a9c5",
      "41e17e10ea1047d565f712447342a2e949dafa0b9e"
    ],
    "offchainConfigVersion": 2,
    "offchainConfig": {
      "deltaProgress": "23s",
      "deltaResend": "12s",
      "deltaRound": "10s",
      "deltaGrace": "1s",
      "deltaStage": "20s",
      "rMax": 6,
      "s": [1, 1, 1, 1],
      "peerIds": [
        "12D3KooWNegwGR5oTKZ3MHL7xe49he4rzT8hxH9oEUQqtYkbBXwK",
        "12D3KooWFcsmakYqgSf91zzJFX158fbPnXa7vJVce7GLNzvFoK4c",
        "12D3KooWAyZYUyS6yNVr92XLn7J5xQpEh3fqLsNPbwy8XCHsK7H3",
        "12D3KooWMFCW4JaXFVcYVn2BLxh6mWF9SqP9gkYJ54ZbGkX18ehG"
      ],
      "maxDurationQuery": "500ms",
      "maxDurationObservation": "5s",
      "maxDurationReport": "10s",
      "maxDurationShouldAcceptFinalizedReport": "10s",
      "maxDurationShouldTransmitAcceptedReport": "10s",
      "reportingPluginConfig": {
        "alphaReportInfinite": false,
        "alphaReportPpb": "5000000",
        "alphaAcceptInfinite": false,
        "alphaAcceptPpb": "5000000",
        "deltaC": "24h"
      },
      "configEncodingSecret": "test secret",
      "signerSecret": "test secret",
      "offchainPublicKeys": [
        "ocr2off_tron_c064dde2f568dde4b6d54929ccd145f45460abd923f009bbe6e8901007a7ba5c",
        "ocr2off_tron_957b7eec9f7a481ad1e3a8ae3080a39f79a7b447e0b85e0b38359cdbfca4704c",
        "ocr2off_tron_24bab7509e07686217293850d12c5d3988546d7cee3e73ef3b9c35266b3b3241",
        "ocr2off_tron_213ca1374814777ef80c9fc84006de0c768e8e0e0dd1be537be31109502c3561"
      ],
      "configPublicKeys": [
        "ocr2cfg_tron_6da430ffde817a7d932588e7e02c31f333460d5ce2b4f3ac79e7802284567b5e",
        "ocr2cfg_tron_0d961d8b4824d3b44ef4d36019137ddc701422d746ec2b3d94e1b82af2f59d00",
        "ocr2cfg_tron_bed4f66fc0ff85f41a6e3083346edc0df779cc360e72b7f16b5018443e567a22",
        "ocr2cfg_tron_26f449bf6a35edb6d2c1489abef1f8167cbf9839990f1c74f7238e2c4e64a379"
      ]
    }
  }
}
```

#### Command

```sh
yarn gauntlet execute \
  -o tron/data-feeds/aggregator:set-config \
  -c config.json \
  -i input.json
```

#### RDD

- After you run this command, note the block number that contains the `set_config` transaction

### Update RDD

#### Aggregator JSON File

- Create a file named `<AGGREGATOR_ADDRESS>.json` in the directory for your environment (e.g. for staging this would be `tron-testnet-nile/contracts/`)
- In the file, copy/paste the template below, then modify the following fields:
  - `config.reportingPluginConfig.alphaAcceptPpb` should be set to 3000000 for stablecoins
  - `config.reportingPluginConfig.alphaReportPpb` should be set to 3000000 for stablecoins
  - `ExternalAdapterRequestParams.from` should be set accordingly based on the input provided to the deploy aggregator operation
  - `lastConfigDigest` should be updated such that it uses the value obtained from the tron explorer - view the contract in the explorer and call the `latestConfigDetails` view function to obtain the value
  - `fromBlock` should be set to the block number which refers to the first `set_config` transaction executed on the contract
  - `marketing.pair` should be updated accordingly
  - `marketing.path` should be updated accordingly
  - `name` should be updated accordingly

```json
{
  "ExternalAdapterRequestParams": {
    "endpoint": "crypto",
    "from": "TRX",
    "to": "USD"
  },
  "billing": {
    "accountingGas": "0",
    "maximumGasPriceGwei": "0",
    "observationPaymentGjuels": "0",
    "reasonableGasPriceGwei": "0",
    "transmissionPaymentGjuels": "0"
  },
  "config": {
    "deltaGrace": "1s",
    "deltaProgress": "23s",
    "deltaResend": "12s",
    "deltaRound": "10s",
    "deltaStage": "20s",
    "f": 1,
    "lastConfigDigest": "0x00018fe3c436f609a0da24b8b3bc54a3b604514300569db9660855af0a1f9c65",
    "maxDurationObservation": "5s",
    "maxDurationQuery": "500ms",
    "maxDurationReport": "10s",
    "maxDurationShouldAcceptFinalizedReport": "10s",
    "maxDurationShouldTransmitAcceptedReport": "10s",
    "rMax": 6,
    "reportingPluginConfig": {
      "alphaAcceptInfinite": false,
      "alphaAcceptPpb": "5000000",
      "alphaReportInfinite": false,
      "alphaReportPpb": "5000000",
      "deltaC": "24h"
    },
    "s": [1, 1, 1, 1]
  },
  "contractVersion": 6,
  "decimals": 18,
  "fromBlock": 54268518,
  "jobSpecOverrides": {
    "juelsPerFeeCoin": 0
  },
  "marketing": {
    "category": "crypto",
    "history": true,
    "pair": ["TRX", "USD"],
    "path": "trx-usd-staging"
  },
  "maxSubmissionValue": "95780971304118053647396689196894323976171195136475135",
  "minSubmissionValue": "1",
  "name": "TRX / USD - Staging",
  "networkShortname": "tron",
  "oracles": [
    {
      "api": ["ncfx", "gsr", "cfbenchmarks"],
      "operator": "ocr2-internal-0"
    },
    {
      "api": ["coinmetrics", "tiingo", "dar"],
      "operator": "ocr2-internal-1"
    },
    {
      "api": ["blocksize-capital", "cryptocompare", "finage"],
      "operator": "ocr2-internal-2"
    },
    {
      "api": ["coinmarketcap", "cfbenchmarks", "tiingo"],
      "operator": "ocr2-internal-3"
    }
  ],
  "status": "live",
  "type": "numerical_median_feed"
}
```

#### Aggregator Proxy JSON File

- Create a file named `<AGGREGATOR_PROXY_ADDRESS>.json` in the directory for your environment (e.g. for staging this would be `tron-testnet-nile/proxies/`)
- In the file, copy/paste the template below, then make the following adjustments:
  - Update all the `<..._ADDRESS>` templates accordingly
  - `name` should be updated accordingly

```json
{
  "accessController": "<ACCESS_CONTROLLER_ADDRESS>",
  "aggregator": "<AGGREGATOR_ADDRESS>",
  "contractVersion": 6,
  "implementations": {
    "backup": "0x0000000000000000000000000000000000000000",
    "primary": "<AGGREGATOR_ADDRESS>",
    "proposed": "0x0000000000000000000000000000000000000000",
    "secondary": "0x0000000000000000000000000000000000000000"
  },
  "name": "ETH / USD - Staging"
}
```

#### Finalize The Updates

In the root directory of the RDD repo, run the following commands:

```sh
# Make sure you have the correct rddtool version
./bin/install_rddtool

# Make sure all RDD files are up to date
./bin/generate && ./bin/degenerate
```

### Inspecting On-Chain State

To inspect the state of an Aggregator contract, you can run:

```sh
yarn gauntlet query \
  -o tron/data-feeds/aggregator:inspect \
  -c config.json \
  -i '{ "address": "<AGGREGATOR_ADDRESS>" }'
```

To inspect the state of an Aggregator Proxy contract, you can run:

```sh
yarn gauntlet query \
  -o tron/data-feeds/aggregator-proxy:inspect \
  -c config.json \
  -i '{ "address": "<AGGREGATOR_PROXY_ADDRESS>" }'
```
