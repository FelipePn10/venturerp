package request

type UpdateScheduleStatusDTO struct {
	Status      string  `json:"status"`
	ProducedQty float64 `json:"produced_qty"`
}

type UpdateScheduleTimesDTO struct {
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
}
