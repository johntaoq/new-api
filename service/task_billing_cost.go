package service

import (
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

func estimateTaskQuotaCostUSD(task *model.Task, quota int) float64 {
	if quota <= 0 {
		return 0
	}
	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}
	return model.BuildRelayEstimatedCostUSD(&relaycommon.RelayInfo{
		PriceData: types.PriceData{
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: groupRatio,
			},
		},
	}, quota)
}
