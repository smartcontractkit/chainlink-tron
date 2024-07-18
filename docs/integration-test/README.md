# Tron OCR2 Integration Test

## Running Local tests

1. Clone and checkout [chainlink-tron](https://github.com/smartcontractkit/chainlink-tron/tree/feat/BCI-2927-TronKeystore) repo pointed to the `feat/BCI-2927-TronKeystore`
2. Symlink the `chainlink-tron` repo to `chainlink-internal-integrations/` directory. (optional)

This step is optional if you have the `chainlink-tron` repo in the same directory as `chainlink-internal-integrations/`

```shell
ln -s <dir/chainlink-tron>
```

3. Checkout `chainlink-internal-integrations` on branch `develop`
4. cd to `chainlink-internal-integrations/tron/integration_tests`

```shell
cd /tron/integration_tests
```

5. Run this command to build the chainlink-tron image and run the e2e test

```shell
   pushd ../../chainlink-tron && ./tron-build.sh && popd && TEST_LOG_LEVEL=debug CORE_IMAGE=chainlink-tron go test -v -tags=integration,testnet -count=1 -run TestOCRLocal ./ocr2_local_test.go
```

6. To know if your test is successful, you should see the following logs:

```shell
INF Transmission Details: {Digest:0001a1a988f75da34469ae9e5bbef908ee33d1855c9ea9c835d751b4296705e4 Epoch:2 Round:1 LatestAnswer:+5 LatestTimestamp:2024-07-07 06:24:36 +0400 +04}
```

## Running Testnet tests

### Shasta Testnet

1. Export private key and make sure you have enough TRX in the wallet. Each OCR2 test on test net requires 7k TRX.

```
export PRIVATE_KEY=<private>
```

2. Follow the same following steps as local test as above (Steps 2 - 6]) but with the following command to replace step 5

Shasta:

```shell
pushd ../../../chainlink-tron && ./tron-build.sh && popd && TEST_LOG_LEVEL=debug CORE_IMAGE=chainlink-tron go test -v -tags=integration,testnet -count=1 -run TestOCR2Shasta ./ocr2_testnet_test.go -timeout 60m | tee test.log
```

Nile:

```shell
ushd ../../../chainlink-tron && ./tron-build.sh && popd && TEST_LOG_LEVEL=debug CORE_IMAGE=chainlink-tron go test -v -tags=integration,testnet -count=1 -run TestOCR2Nile ./ocr2_testnet_test.go -timeout 60m | tee test.log
```

### Pre-requisites

- Docker v4.25.2 (Latest versions of docker do not work. See [here](https://smartcontract-it.atlassian.net/wiki/spaces/DEPLOY/pages/774734068/Tron+Node+Errors+And+Fixes))

---
