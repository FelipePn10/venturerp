package request

type ProductionPlanInterFactoryDTO struct {
	EnterpriseCode int64 `json:"enterprise_code"`
	AutoRelease    bool  `json:"auto_release"`
}

type ReplaceProductionPlanInterFactoriesDTO struct {
	Enterprises []ProductionPlanInterFactoryDTO `json:"enterprises"`
}
