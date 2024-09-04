package jsonclient

var expectedGetAccount = &GetAccountResponse{
	FrozenV2: []Account_FreezeV2{
		{},
		{
			Type: "ENERGY",
		},
		{
			Type: "TRON_POWER",
		},
	},
	Balance:               5994521520,
	LatestOprationTime:    1715145693000,
	LatestConsumeFreeTime: 1715145693000,
	NetWindowOptimized:    true,
	AccountResource: AccountResource{
		LatestConsumeTimeForEnergy: 1717675329000,
		EnergyWindowSize:           28800000,
		EnergyWindowOptimized:      true,
	},
	ActivePermission: []Permission{
		{
			Keys: []*Key{
				{
					Address: "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
					Weight:  1,
				},
			},
			Type:           "Active",
			ID:             2,
			PermissionName: "active",
			Threshold:      1,
			Operations:     "7fff1fc0033e0300000000000000000000000000000000000000000000000000",
		},
	},
	Address:           "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
	CreateTime:        1637411046000,
	LatestConsumeTime: 1717666029000,
	NetWindowSize:     28800000,
	OwnerPermission: Permission{
		Threshold: 1,
		Keys: []*Key{
			{
				Address: "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
				Weight:  1,
			},
		},
		PermissionName: "owner",
	},
}
