package types

type OrderType string

const (
	OrderProduction          OrderType = "PRODUCTION"
	OrderPurchase            OrderType = "PURCHASE"
	OrderOutsourcing         OrderType = "OUTSOURCING"
	OrderTechnicalAssistance OrderType = "TECHNICAL_ASSISTANCE"
)

type OrderStatus string

const (
	StatusPlanned    OrderStatus = "PLANNED"
	StatusReleased   OrderStatus = "RELEASED"
	StatusInProgress OrderStatus = "IN_PROGRESS"
	StatusFinished   OrderStatus = "FINISHED"
	StatusCancelled  OrderStatus = "CANCELLED"
)
