package sqltypes

import (
	"database/sql/driver"
	"fmt"
)

// ─── OperationOriginEnum ──────────────────────────────────────────────────────

type OperationOriginEnum string

const (
	OperationOriginEnumINTERNA   OperationOriginEnum = "INTERNA"
	OperationOriginEnumEXTERNA   OperationOriginEnum = "EXTERNA"
	OperationOriginEnumTERCEIROS OperationOriginEnum = "TERCEIROS"
)

func (e *OperationOriginEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = OperationOriginEnum(s)
	case string:
		*e = OperationOriginEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for OperationOriginEnum: %T", src)
	}
	return nil
}

type NullOperationOriginEnum struct {
	OperationOriginEnum OperationOriginEnum
	Valid               bool
}

func (ns *NullOperationOriginEnum) Scan(value interface{}) error {
	if value == nil {
		ns.OperationOriginEnum, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.OperationOriginEnum.Scan(value)
}

func (ns NullOperationOriginEnum) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.OperationOriginEnum), nil
}

// ─── OperationSituationEnum ───────────────────────────────────────────────────

type OperationSituationEnum string

const (
	OperationSituationEnumAPROVADA OperationSituationEnum = "APROVADA"
	OperationSituationEnumINATIVA  OperationSituationEnum = "INATIVA"
)

func (e *OperationSituationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = OperationSituationEnum(s)
	case string:
		*e = OperationSituationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for OperationSituationEnum: %T", src)
	}
	return nil
}

type NullOperationSituationEnum struct {
	OperationSituationEnum OperationSituationEnum
	Valid                  bool
}

func (ns *NullOperationSituationEnum) Scan(value interface{}) error {
	if value == nil {
		ns.OperationSituationEnum, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.OperationSituationEnum.Scan(value)
}

func (ns NullOperationSituationEnum) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.OperationSituationEnum), nil
}

// ─── RouteSituationEnum ───────────────────────────────────────────────────────

type RouteSituationEnum string

const (
	RouteSituationEnumAPROVADA RouteSituationEnum = "APROVADA"
	RouteSituationEnumINATIVA  RouteSituationEnum = "INATIVA"
)

func (e *RouteSituationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = RouteSituationEnum(s)
	case string:
		*e = RouteSituationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for RouteSituationEnum: %T", src)
	}
	return nil
}

type NullRouteSituationEnum struct {
	RouteSituationEnum RouteSituationEnum
	Valid              bool
}

func (ns *NullRouteSituationEnum) Scan(value interface{}) error {
	if value == nil {
		ns.RouteSituationEnum, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.RouteSituationEnum.Scan(value)
}

func (ns NullRouteSituationEnum) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.RouteSituationEnum), nil
}

// ─── RouteOpSituationEnum ─────────────────────────────────────────────────────

type RouteOpSituationEnum string

const (
	RouteOpSituationEnumAPROVADA RouteOpSituationEnum = "APROVADA"
	RouteOpSituationEnumINATIVA  RouteOpSituationEnum = "INATIVA"
	RouteOpSituationEnumFANTASMA RouteOpSituationEnum = "FANTASMA"
)

func (e *RouteOpSituationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = RouteOpSituationEnum(s)
	case string:
		*e = RouteOpSituationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for RouteOpSituationEnum: %T", src)
	}
	return nil
}

type NullRouteOpSituationEnum struct {
	RouteOpSituationEnum RouteOpSituationEnum
	Valid                bool
}

func (ns *NullRouteOpSituationEnum) Scan(value interface{}) error {
	if value == nil {
		ns.RouteOpSituationEnum, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.RouteOpSituationEnum.Scan(value)
}

func (ns NullRouteOpSituationEnum) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.RouteOpSituationEnum), nil
}
