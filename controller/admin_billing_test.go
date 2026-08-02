package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestResolveQuotaAdjustmentSourceTypePaidReasons(t *testing.T) {
	reasons := []string{
		model.QuotaFundingSourceCompensation,
		model.QuotaFundingSourceRefund,
		model.QuotaFundingSourceManualTopUp,
		model.QuotaFundingSourceAccountClosure,
	}
	for _, reason := range reasons {
		t.Run(reason, func(t *testing.T) {
			resolved, err := resolveQuotaAdjustmentSourceType(model.QuotaFundingTypePaid, 1, reason, true)
			require.NoError(t, err)
			require.Equal(t, reason, resolved)
		})
	}
}

func TestResolveQuotaAdjustmentSourceTypeRejectsInvalidPaidReason(t *testing.T) {
	_, err := resolveQuotaAdjustmentSourceType(model.QuotaFundingTypePaid, 1, "", true)
	require.ErrorContains(t, err, "reason is required")

	_, err = resolveQuotaAdjustmentSourceType(model.QuotaFundingTypePaid, 1, "system_adjustment", true)
	require.ErrorContains(t, err, "invalid paid quota adjustment reason")

	_, err = resolveQuotaAdjustmentSourceType(model.QuotaFundingTypePaid, 1, model.QuotaFundingSourceCompensation, false)
	require.ErrorContains(t, err, "only root")
}

func TestResolveQuotaAdjustmentEntryType(t *testing.T) {
	require.Equal(t, model.CustomerMonthlyStatementEntryTypeTopup,
		resolveQuotaAdjustmentEntryType(model.QuotaFundingTypePaid, 1, model.QuotaFundingSourceManualTopUp))
	require.Equal(t, model.CustomerMonthlyStatementEntryTypeAdjustment,
		resolveQuotaAdjustmentEntryType(model.QuotaFundingTypePaid, 1, model.QuotaFundingSourceCompensation))
	require.Equal(t, model.CustomerMonthlyStatementEntryTypeAdjustment,
		resolveQuotaAdjustmentEntryType(model.QuotaFundingTypePaid, -1, model.QuotaFundingSourceManualTopUp))
	require.Equal(t, model.CustomerMonthlyStatementEntryTypeGift,
		resolveQuotaAdjustmentEntryType(model.QuotaFundingTypeGift, 1, model.QuotaFundingSourceAdminGrant))
}
