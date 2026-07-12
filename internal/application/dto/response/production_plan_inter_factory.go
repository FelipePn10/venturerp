package response

type ProductionPlanInterFactoryResponse struct {
	EnterpriseCode int64  `json:"enterprise_code"`
	EnterpriseName string `json:"enterprise_name"`
	AutoRelease    bool   `json:"auto_release"`
}
