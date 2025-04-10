# Deploy CCIP contracts

## Tron

Quick overview on how to deploy CCIP contracts on Tron:

```bash
yarn gauntlet execute -o tron/token/burnable-link:deploy --config ./config.json
yarn gauntlet execute -o tron/token/burnable-link:grant-mint-burn-role --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/token/burnable-link:mint --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/rmn@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/rmn-proxy@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/router@1.2.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/token-admin-registry@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/registry-module-owner-custom@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/token-admin-registry@1.5.0:add-registry-module --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/lock-release-token-pool@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/lock-release-token-pool@1.5.0:set-rebalancer --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/evm2evm-on-ramp@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/router@1.2.0:apply-ramp-updates --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/token-pool@1.5.0:apply-chain-updates --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/commit-store@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:apply-price-updaters-updates --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/evm2evm-off-ramp@1.5.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/router@1.2.0:apply-ramp-updates --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/rmn@1.5.0:owner-remove-then-add-perma-blessed-commit-stores --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:update-prices --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/ping-pong-demo@1.4.0:deploy --config ./config.json --input ./input.json
yarn gauntlet execute -o tron/ccip/ping-pong-demo@1.4.0:set-counterpart --config ./config.json --input ./input.json
```

## Ethereum

Quick overview on how to deploy CCIP contracts on Ethereum:

```bash
TODO!
```
