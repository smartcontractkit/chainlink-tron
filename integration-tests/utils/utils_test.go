package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateOCR2Report(t *testing.T) {
	expectedReport := "0x000000000000000000000000000000000000000000000000000000007d03946f0e04000f14050b1b1e1d0d121607101117080c1902091c15011a060a1813030000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000000000000000000000001f000000000000000000000000000000000000000000000000077a2d8f5e91f0270000000000000000000000000000000000000000000000000ae887525a3f15e500000000000000000000000000000000000000000000000011a3f35609c4d14d00000000000000000000000000000000000000000000000019a602f096065cf00000000000000000000000000000000000000000000000001ad6d700173e6e5400000000000000000000000000000000000000000000000025e869828d14179c0000000000000000000000000000000000000000000000002780cd1336c7d51900000000000000000000000000000000000000000000000039eb1560491577a80000000000000000000000000000000000000000000000003f619a6d2f1057930000000000000000000000000000000000000000000000003f7f9a043011bfae00000000000000000000000000000000000000000000000043fa06cc4d94a0b800000000000000000000000000000000000000000000000047ad7eae88419beb00000000000000000000000000000000000000000000000047ae091029c935af000000000000000000000000000000000000000000000000502a6cb672755ae400000000000000000000000000000000000000000000000053e55047cdedc27b00000000000000000000000000000000000000000000000055eb22a907d120e800000000000000000000000000000000000000000000000055f31dc14d6a5a5e000000000000000000000000000000000000000000000000560e1f4cfd107fe10000000000000000000000000000000000000000000000005763e0dcd3d1d9f90000000000000000000000000000000000000000000000005a63caf7c23fb8f20000000000000000000000000000000000000000000000005c0c78111b9f968e0000000000000000000000000000000000000000000000005cffcef5d3b7c2ce0000000000000000000000000000000000000000000000005d8a2152bc4bfe4000000000000000000000000000000000000000000000000060fda75a4e7a6b0d000000000000000000000000000000000000000000000000632089ffe2da8f1e0000000000000000000000000000000000000000000000006c72eaf17a4df8490000000000000000000000000000000000000000000000006dcb9d28a62a74fe000000000000000000000000000000000000000000000000712b8b76beb6aa8b00000000000000000000000000000000000000000000000072183178f7f17ae200000000000000000000000000000000000000000000000077e64109490884830000000000000000000000000000000000000000000000007bad7fe9466ebdda"

	expectedMedianReportValue := "0x55eb22a907d120e8"

	report, medianReportValue, err := GenerateOCR2Report()
	require.NoError(t, err)
	require.Equal(t, expectedReport, report)
	require.Equal(t, expectedMedianReportValue, medianReportValue)
}

func TestFunctionSignatureHash(t *testing.T) {
	expectedHash := "0x1591690b8638f5fb2dbec82ac741805ac5da8b45dc5263f4875b0496fdce4e05"
	eventSignature := "ConfigSet(uint32,bytes32,uint64,address[],address[],uint8,bytes,uint64,bytes)"
	hash := FunctionSignatureHash(eventSignature)
	require.Equal(t, expectedHash, hash)
}
