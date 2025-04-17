# G++ Testing Wiki

## Networks

### Mainnet

- Chain Name: Tron Mainnet
- Chain Id: `0x2b6653dc` - last four bytes of the genesis block hash
- Public RPCs: [https://api.trongrid.io/](https://api.trongrid.io/)

Find more details in tron docs [here](https://developers.tron.network/docs/networks). We are using the following URLs for testing [here](https://github.com/smartcontractkit/chainlink-internal-integrations/blob/16f8e7e5e1749f7d6542b641b0d60bc8b38cec8f/tron/relayer/testutils/tron_node.go#L11).

### Shasta

Description: Shasta is the testnet that is used for testing purposes. The parameters of the Shasta testnet are consistent with the mainnet.

- Chain Name: [Shasta Testnet](https://shasta.tronscan.org/#/)
- Chain Id: `0x94a9059e` - last four bytes of the genesis block hash
- Public RPCs: [https://api.shasta.trongrid.io/](https://api.shasta.trongrid.io/)

### Nile

- Not included in the testing, as Nile testnet is used to test new features of TRON, and the code version is generally ahead of the mainnet.

However the configuration can be found [here](https://developers.tron.network/docs/networks)

### Node Config

```toml
[[Tron]]
Enabled = true
ChainID = '<id>'

[[Tron.Nodes]]
Name = 'primary'
URL = '<full node http url>'
SolidityURL = '<solidity http url>'

[Log]
Level = 'debug'

[OCR2]
Enabled = true

[P2P]
[P2P.V2]
Enabled = true
DeltaDial = '5s'
DeltaReconcile = '5s'
ListenAddresses = ['0.0.0.0:6691']

[WebServer]
HTTPPort = 6688
[WebServer.TLS]
HTTPSPort = 0
`
```

## Gauntlet steps

### Setup ENV file

```toml
NODE_URL=<rpc_url>
ACCOUNT=<account>
PRIVATE_KEY=<private_key>
CHAINLINK_ENV_USER=John;
CHAINLINK_IMAGE={AWS_OIDC}.dkr.ecr.{AWS_REGION}.amazonaws.com/chainlink;
CHAINLINK_VERSION=develop;
INTERNAL_DOCKER_REPO={AWS_OIDC}.dkr.ecr.{AWS_REGION}.amazonaws.com; # required for mock adapter
RPC_URL=; # testnet only
NODE_COUNT=5;
TEST_DURATION=70h; # for soak
TEST_USE_ENV_VAR_CONFIG=true; # for soak
TTL=72h # for soak
```

### Deployment of contracts

1. Deploy link

```bash
yarn gauntlet token:deploy --link
```

2. Deploy access controller

```bash
yarn gauntlet access_controller:deploy
```

3. Deploy OCR2

```bash
yarn gauntlet ocr2:deploy --minSubmissionValue=<value> --maxSubmissionValue=<value> --decimals=<value> --name=<value> --link=<link_addr>
```

4. Deploy proxy

```bash
yarn gauntlet proxy:deploy <ocr_address>
```

5. Add access to proxy

```bash
yarn gauntlet ocr2:add_access --address=<ocr_address> <proxy_address>
```

6. Mint LINK

```bash
yarn gauntlet token:mint --recipient<ocr_addr> --amount=<value> <link_addr>
```

7. Set billing

```bash
yarn gauntlet ocr2:set_billing --observationPaymentGjuels=<value> --transmissionPaymentGjuels=<value> <ocr_addr>
```

8. Set config

   1. Example config testnet

   ```bash
   {
       "f": 1,
       "signers": [
           "ocr2on_starknet_0371028377bfd793b7e2965757e348309e7242802d20253da6ab81c8eb4b4051",
           "ocr2on_starknet_073cadfc4474e8c6c79f66fa609da1dbcd5be4299ff9b1f71646206d1faca1fc",
           "ocr2on_starknet_0386d1a9d93792c426739f73afa1d0b19782fbf30ae27ce33c9fbd4da659cd80",
           "ocr2on_starknet_005360052758819ba2af790469a28353b7ff6f8b84176064ab572f6cc20e5fb4"
       ],
       "transmitters": [
           "0x0...",
           "0x0...",
           "0x0...",
           "0x0..."
       ],
       "onchainConfig": "",
       "offchainConfig": {
           "deltaProgressNanoseconds": 8000000000,
           "deltaResendNanoseconds": 30000000000,
           "deltaRoundNanoseconds": 3000000000,
           "deltaGraceNanoseconds": 1000000000,
           "deltaStageNanoseconds": 20000000000,
           "rMax": 5,
           "s": [
               1,
               1,
               1,
               1
           ],
           "offchainPublicKeys": [
               "ocr2off_starknet_0...",
               "ocr2off_starknet_0...",
               "ocr2off_starknet_0...",
               "ocr2off_starknet_0..."
           ],
           "peerIds": [
               "12D3..",
               "12D3..",
               "12D3..",
               "12D3.."
           ],
           "reportingPluginConfig": {
               "alphaReportInfinite": false,
               "alphaReportPpb": 0,
               "alphaAcceptInfinite": false,
               "alphaAcceptPpb": 0,
               "deltaCNanoseconds": 1000000000
           },
           "maxDurationQueryNanoseconds": 2000000000,
           "maxDurationObservationNanoseconds": 1000000000,
           "maxDurationReportNanoseconds": 2000000000,
           "maxDurationShouldAcceptFinalizedReportNanoseconds": 2000000000,
           "maxDurationShouldTransmitAcceptedReportNanoseconds": 2000000000,
           "configPublicKeys": [
               "ocr2cfg_starknet_...",
               "ocr2cfg_starknet_...",
               "ocr2cfg_starknet_...",
               "ocr2cfg_starknet_..."
           ]
       },
       "offchainConfigVersion": 2,
       "secret": "some secret you want"
   }
   ```

```bash
yarn gauntlet ocr2:set_config --input=<cfg> <ocr_addr>
```
