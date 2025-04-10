yarn gauntlet execute -o tron/token/burnable-link:deploy --config ./config.json

yarn gauntlet execute -o tron/token/burnable-link:grant-mint-burn-role --config ./config.json --input ./input.json
`{"address": "link-address","account": "account-to-grant-permissions-to"}`

yarn gauntlet execute -o tron/token/burnable-link:mint --config ./config.json --input ./input.json
`{"linkTokenAddress": "link-address","account": "account-to-grant-permissions-to", "amount": "The amount of tokens to mint"}`

yarn gauntlet execute -o tron/ccip/rmn@1.5.0:deploy --config ./config.json --input ./input.json
`{"voters": [{"blessWeight": 1, "curseWeight": 1, "blessVoteAddr": "any-random-address", "curseVoteAddr": "any-random-address"}], "blessWeightThreshold": 1, "curseWeightThreshold": 1}`

yarn gauntlet execute -o tron/ccip/rmn-proxy@1.5.0:deploy --config ./config.json --input ./input.json
`{ "rmnAddress": "rmn-address" }`

yarn gauntlet execute -o tron/ccip/router@1.2.0:deploy --config ./config.json --input ./input.json
`{ "wrappedNative": "41fb3b3134F13CcD2C81F4012E53024e8135d58FeE", "armProxy": "rmn-proxy" }`

yarn gauntlet execute -o tron/ccip/token-admin-registry@1.5.0:deploy --config ./config.json

yarn gauntlet execute -o tron/ccip/registry-module-owner-custom@1.5.0:deploy --config ./config.json --input ./input.json
`{ "tokenAdminRegistry": "token-admin-registry" }`

yarn gauntlet execute -o tron/ccip/token-admin-registry@1.5.0:add-registry-module --config ./config.json --input ./input.json
`{ "address": "token-admin-registry-address", "registryModuleAddress": "registry-module-address" }`

yarn gauntlet execute -o tron/ccip/lock-release-token-pool@1.5.0:deploy --config ./config.json --input ./input.json
`{ "token": "link-token-address", "allowList": [], "rmnProxy": "rmn-proxy-address", "acceptLiquidity": true, "router": "router-address" }`

yarn gauntlet execute -o tron/ccip/lock-release-token-pool@1.5.0:set-rebalancer --config ./config.json --input ./input.json
`{ "address": "token-pool-address", "rebalancer": "deployer-address" }`

yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:deploy --config ./config.json --input ./input.json
`{ "priceUpdaters": [], "feeTokens": ["link-token-address", "wrapped-native"], "stalenessThreshold": 1209600 }`

yarn gauntlet execute -o tron/ccip/evm2evm-on-ramp@1.5.0:deploy --config ./config.json --input ./input.json
`{
	"staticConfig": {
		"linkToken": "link-token-address",
		"chainSelector": "source-chain-selector",
		"destChainSelector": "dest-chain-selector",
		"defaultTxGasLimit": "200000",
		"maxNopFeesJuels": "100000000000000000000000000",
		"prevOnRamp": "410000000000000000000000000000000000000000",
		"rmnProxy": "rmn-proxy-address",
		"tokenAdminRegistry": "token-admin-registry-address"
	},
	"dynamicConfig": {
		"router": "router-address",
		"maxNumberOfTokensPerMsg": 1,
		"destGasOverhead": "350000",
		"destGasPerPayloadByte": "16",
		"destDataAvailabilityOverheadGas": "33596",
		"destGasPerDataAvailabilityByte": "16",
		"destDataAvailabilityMultiplierBps": "6840",
		"priceRegistry": "price-registry-address",
		"maxDataBytes": "30000",
		"maxPerMsgGasLimit": "3000000",
		"defaultTokenFeeUSDCents": "50",
		"defaultTokenDestGasOverhead": "90000",
		"enforceOutOfOrder": false
	},
	"rateLimiterConfig": {
		"isEnabled": false,
		"capacity": 0,
		"rate": 0
	},
	"feeTokenConfigArgs": [
		{
			"token": "link-token-address",
			"networkFeeUSDCents": "10",
			"gasMultiplierWeiPerEth": "100000000000000000",
			"premiumMultiplierWeiPerEth": "900000000000000000",
			"enabled": true
		},
		{
			"token": "41fb3b3134F13CcD2C81F4012E53024e8135d58FeE",
			"networkFeeUSDCents": "10",
			"gasMultiplierWeiPerEth": "1000000000000000000",
			"premiumMultiplierWeiPerEth": "1000000000000000000",
			"enabled": true
		}
	],
	"tokenTransferFeeConfigArgs": [
		{
			"token": "link-token-address",
			"minFeeUSDCents": "10",
			"maxFeeUSDCents": "5000",
			"deciBps": "0",
			"destGasOverhead": "90000",
			"destBytesOverhead": "32",
			"aggregateRateLimitEnabled": false
		},
		{
			"token": "41fb3b3134F13CcD2C81F4012E53024e8135d58FeE",
			"minFeeUSDCents": "10",
			"maxFeeUSDCents": "5000",
			"deciBps": "10",
			"destGasOverhead": "90000",
			"destBytesOverhead": "32",
			"aggregateRateLimitEnabled": true
		}
	],
	"nopsAndWeights": []
}
`

yarn gauntlet execute -o tron/ccip/router@1.2.0:apply-ramp-updates --config ./config.json --input ./input.json
`{"address": "router-address", "onRampUpdates": [{ "destChainSelector": "destination-chain-selector", "onRamp": "on-ramp" }], "offRampRemoves": [], "onRampAdds": []}`

yarn gauntlet execute -o tron/ccip/token-pool@1.5.0:apply-chain-updates --config ./config.json --input ./input.json
`
{
	"address": "token-pool-address",
	"chains": [
		{
			"remotePoolAddress": "remote-token-pool",
			"remoteTokenAddress": "remote-link",
			"remoteChainSelector": "destination-chain-selector",
			"allowed": true,
			"inboundRateLimiterConfig": { "isEnabled": false, "rate": 0, "capacity": 0 },
			"outboundRateLimiterConfig": { "isEnabled": false, "rate": 0, "capacity": 0 }
		}
	]
}
`

yarn gauntlet execute -o tron/ccip/commit-store@1.5.0:deploy --config ./config.json --input ./input.json
`{ "staticConfig": { "chainSelector": "destination-selector", "sourceChainSelector": "source-selector", "onRamp": "", "rmnProxy": "" } }`

yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:apply-price-updaters-updates --config ./config.json --input ./input.json
`{ "address": "price-registry", "priceUpdatersToAdd": ["commitstore-address"], "priceUpdatersToRemove": [] }`

yarn gauntlet execute -o tron/ccip/evm2evm-off-ramp@1.5.0:deploy --config ./config.json --input ./input.json
`
{
	"staticConfig": {
		"commitStore": "source-commit-store",
		"chainSelector": "destination-chain-selector",
		"sourceChainSelector": "source-chain-selector",
		"onRamp": "source-on-ramp",
		"prevOffRamp": "410000000000000000000000000000000000000000",
		"rmnProxy": "rmn-proxy",
		"tokenAdminRegistry": "token-admin-registry",
	},
	"rateLimiterConfig": {
		"isEnabled": false,
		"capacity": 0,
		"rate": 0
	}
}
`

yarn gauntlet execute -o tron/ccip/router@1.2.0:apply-ramp-updates --config ./config.json --input ./input.json
`
{
	"address": "router-address",
	"onRampUpdates": [],
	"offRampRemoves": [],
	"onRampAdds": [
		{
			"sourceChainSelector": "source-chain-selector",
			"offRamp": "off-ramp"
		}
	]
}
`

yarn gauntlet execute -o tron/ccip/rmn@1.5.0:owner-remove-then-add-perma-blessed-commit-stores --config ./config.json --input ./input.json
`{ "address": "rmn-address", "removes": [], "adds": ["commit-store"] }`

yarn gauntlet execute -o tron/ccip/price-registry@1.2.0:update-prices --config ./config.json --input ./input.json
`
{
	"address": "price-registry",
	"priceUpdates": {
		"tokenPriceUpdates": [
			{
				"sourceToken": "source-link-address",
				"usdPerToken": "10000000000000000000"
			},
			{
				"sourceToken": "source-wrapped-native",
				"usdPerToken": "2000000000000000000000"
			}
		],
		"gasPriceUpdates": [{ "destChainSelector": "dest-selector", "usdPerUnitGas": "2000000000000" }]
	}
}
`

yarn gauntlet execute -o tron/ccip/ping-pong-demo@1.4.0:deploy --config ./config.json --input ./input.json
`{ "routerAddress": "", "tokenAddress": "link-token-address" }`

yarn gauntlet execute -o tron/ccip/ping-pong-demo@1.4.0:set-counterpart --config ./config.json --input ./input.json
`{ "address": "ping-pong-address", "counterpartChainSelector": "destination-selector", "counterpartAddress": "destination-ping-pong-address" }`