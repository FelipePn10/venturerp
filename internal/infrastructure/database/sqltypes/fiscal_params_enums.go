package sqltypes

import (
	"database/sql/driver"
	"fmt"
)

// ─── CfopUtilizationEnum ──────────────────────────────────────────────────────

type CfopUtilizationEnum string

const (
	CfopUtilizationEnumINDUSTRIALIZACAO_COMERCIO CfopUtilizationEnum = "INDUSTRIALIZACAO_COMERCIO"
	CfopUtilizationEnumIMOBILIZADO               CfopUtilizationEnum = "IMOBILIZADO"
	CfopUtilizationEnumUSO_CONSUMO               CfopUtilizationEnum = "USO_CONSUMO"
)

func (e *CfopUtilizationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CfopUtilizationEnum(s)
	case string:
		*e = CfopUtilizationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CfopUtilizationEnum: %T", src)
	}
	return nil
}

func (e CfopUtilizationEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── CfopIndOperacaoEnum ──────────────────────────────────────────────────────

type CfopIndOperacaoEnum string

const (
	CfopIndOperacaoEnumNORMAL           CfopIndOperacaoEnum = "NORMAL"
	CfopIndOperacaoEnumENERGIA_ELETRICA CfopIndOperacaoEnum = "ENERGIA_ELETRICA"
	CfopIndOperacaoEnumTELECOMUNICACAO  CfopIndOperacaoEnum = "TELECOMUNICACAO"
)

func (e *CfopIndOperacaoEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CfopIndOperacaoEnum(s)
	case string:
		*e = CfopIndOperacaoEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CfopIndOperacaoEnum: %T", src)
	}
	return nil
}

func (e CfopIndOperacaoEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── CfopTipoUtilizacaoEnum ───────────────────────────────────────────────────

type CfopTipoUtilizacaoEnum string

const (
	CfopTipoUtilizacaoEnumNORMAL                           CfopTipoUtilizacaoEnum = "NORMAL"
	CfopTipoUtilizacaoEnumVENDA_COMERCIAL_EXPORTADORA      CfopTipoUtilizacaoEnum = "VENDA_COMERCIAL_EXPORTADORA"
	CfopTipoUtilizacaoEnumCOMPRA_FIM_ESPECIFICO_EXPORTACAO CfopTipoUtilizacaoEnum = "COMPRA_FIM_ESPECIFICO_EXPORTACAO"
	CfopTipoUtilizacaoEnumEXPORTACAO                       CfopTipoUtilizacaoEnum = "EXPORTACAO"
)

func (e *CfopTipoUtilizacaoEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CfopTipoUtilizacaoEnum(s)
	case string:
		*e = CfopTipoUtilizacaoEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CfopTipoUtilizacaoEnum: %T", src)
	}
	return nil
}

func (e CfopTipoUtilizacaoEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── TaxParamOperationEnum ────────────────────────────────────────────────────

type TaxParamOperationEnum string

const (
	TaxParamOperationEnumAMBAS   TaxParamOperationEnum = "AMBAS"
	TaxParamOperationEnumENTRADA TaxParamOperationEnum = "ENTRADA"
	TaxParamOperationEnumSAIDA   TaxParamOperationEnum = "SAIDA"
	TaxParamOperationEnumCUSTOS  TaxParamOperationEnum = "CUSTOS"
)

func (e *TaxParamOperationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TaxParamOperationEnum(s)
	case string:
		*e = TaxParamOperationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for TaxParamOperationEnum: %T", src)
	}
	return nil
}

func (e TaxParamOperationEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── IcmsReductionTargetEnum ──────────────────────────────────────────────────

type IcmsReductionTargetEnum string

const (
	IcmsReductionTargetEnumBASE       IcmsReductionTargetEnum = "BASE"
	IcmsReductionTargetEnumPERCENTUAL IcmsReductionTargetEnum = "PERCENTUAL"
)

func (e *IcmsReductionTargetEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = IcmsReductionTargetEnum(s)
	case string:
		*e = IcmsReductionTargetEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for IcmsReductionTargetEnum: %T", src)
	}
	return nil
}

func (e IcmsReductionTargetEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── IcmsDifalTypeEnum ────────────────────────────────────────────────────────

type IcmsDifalTypeEnum string

const (
	IcmsDifalTypeEnumTRIBUTADO     IcmsDifalTypeEnum = "TRIBUTADO"
	IcmsDifalTypeEnumISENTO_OUTRAS IcmsDifalTypeEnum = "ISENTO_OUTRAS"
	IcmsDifalTypeEnumNAO_CONSIDERA IcmsDifalTypeEnum = "NAO_CONSIDERA"
)

func (e *IcmsDifalTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = IcmsDifalTypeEnum(s)
	case string:
		*e = IcmsDifalTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for IcmsDifalTypeEnum: %T", src)
	}
	return nil
}

func (e IcmsDifalTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── IcmsAcresTypeEnum ────────────────────────────────────────────────────────

type IcmsAcresTypeEnum string

const (
	IcmsAcresTypeEnumFUNDO_COMBATE_POBREZA IcmsAcresTypeEnum = "FUNDO_COMBATE_POBREZA"
	IcmsAcresTypeEnumOUTROS                IcmsAcresTypeEnum = "OUTROS"
)

func (e *IcmsAcresTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = IcmsAcresTypeEnum(s)
	case string:
		*e = IcmsAcresTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for IcmsAcresTypeEnum: %T", src)
	}
	return nil
}

func (e IcmsAcresTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}
