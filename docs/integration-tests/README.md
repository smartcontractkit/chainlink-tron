# Tron OCR2 Integration Test

## Running Local tests

1. (Optional) To run tests against a local core node version/commit, build the core node image locally:

```shell
   cd chainlink
   docker build . -t chainlink-develop:latest -f ./core/chainlink.Dockerfile
```

Then, build the core node with the tron relayer:

```shell
   cd chainlink-tron
   docker buildx build --build-arg BASE_IMAGE=chainlink-develop:latest -t chainlink-tron -f ./Dockerfile .
```

Otherwise, you can skip these steps and it will default to the core container version specified in `scripts/core.sh`.

2. Make sure you are in the Tron integration tests directory

```shell
   cd chainlink-tron/integration_tests
```

3. Run the e2e test:

```shell
   # Run tests against specific core node image that you built in step 1
   TEST_LOG_LEVEL=debug CORE_IMAGE=chainlink-tron go test -v -tags=integration -count=1 -timeout 30m -run TestOCRLocal ./ocr2_local_test.go

   # Or run tests against default core container version
   TEST_LOG_LEVEL=debug go test -v -tags=integration -count=1 -timeout 30m -run TestOCRLocal ./ocr2_local_test.go
```

4. To know if your test is successful, you should see the following logs:

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
