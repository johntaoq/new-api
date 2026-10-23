package authz

const ResourceFinance = "finance"

var (
	FinanceRead  = Permission{Resource: ResourceFinance, Action: ActionRead}
	FinanceWrite = Permission{Resource: ResourceFinance, Action: ActionWrite}
	FinanceAudit = Permission{Resource: ResourceFinance, Action: "audit"}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceFinance,
		LabelKey: "Finance Center",
		Actions: []ActionDefinition{
			{Action: ActionRead, LabelKey: "View finance reports", DescriptionKey: "View revenue, channel costs and customer statements."},
			{Action: ActionWrite, LabelKey: "Monthly exchange rate", DescriptionKey: "Next month exchange rate"},
			{Action: "audit", LabelKey: "View financial audit", DescriptionKey: "View financial changes and their before and after snapshots."},
		},
	})
}
