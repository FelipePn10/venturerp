package sqltypes

import (
	"database/sql/driver"
	"fmt"
)

// ─── CustomerCategoryEnum ─────────────────────────────────────────────────────

type CustomerCategoryEnum string

const (
	CustomerCategoryEnumNORMAL     CustomerCategoryEnum = "NORMAL"
	CustomerCategoryEnumCONSUMIDOR CustomerCategoryEnum = "CONSUMIDOR"
)

func (e *CustomerCategoryEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CustomerCategoryEnum(s)
	case string:
		*e = CustomerCategoryEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CustomerCategoryEnum: %T", src)
	}
	return nil
}

func (e CustomerCategoryEnum) Value() (driver.Value, error) {
	return string(e), nil
}

type NullCustomerCategoryEnum struct {
	CustomerCategoryEnum CustomerCategoryEnum
	Valid                bool
}

func (ns *NullCustomerCategoryEnum) Scan(value interface{}) error {
	if value == nil {
		ns.CustomerCategoryEnum, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.CustomerCategoryEnum.Scan(value)
}

func (ns NullCustomerCategoryEnum) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.CustomerCategoryEnum), nil
}

// ─── CarrierBillingTypeEnum ───────────────────────────────────────────────────

type CarrierBillingTypeEnum string

const (
	CarrierBillingTypeEnumCARTEIRA            CarrierBillingTypeEnum = "CARTEIRA"
	CarrierBillingTypeEnumCOBRANCA_ESCRITURAL CarrierBillingTypeEnum = "COBRANCA_ESCRITURAL"
	CarrierBillingTypeEnumBOLETO              CarrierBillingTypeEnum = "BOLETO"
)

func (e *CarrierBillingTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CarrierBillingTypeEnum(s)
	case string:
		*e = CarrierBillingTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CarrierBillingTypeEnum: %T", src)
	}
	return nil
}

func (e CarrierBillingTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── PaymentAnalysisEnum ──────────────────────────────────────────────────────

type PaymentAnalysisEnum string

const (
	PaymentAnalysisEnumSEMPRE_ANALISA     PaymentAnalysisEnum = "SEMPRE_ANALISA"
	PaymentAnalysisEnumBLOQUEIA_SEMPRE    PaymentAnalysisEnum = "BLOQUEIA_SEMPRE"
	PaymentAnalysisEnumLIBERA_SEM_ANALISE PaymentAnalysisEnum = "LIBERA_SEM_ANALISE"
)

func (e *PaymentAnalysisEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PaymentAnalysisEnum(s)
	case string:
		*e = PaymentAnalysisEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for PaymentAnalysisEnum: %T", src)
	}
	return nil
}

func (e PaymentAnalysisEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── PaymentParcelStartEnum ───────────────────────────────────────────────────

type PaymentParcelStartEnum string

const (
	PaymentParcelStartEnumEMISSAO          PaymentParcelStartEnum = "EMISSAO"
	PaymentParcelStartEnumPROXIMO_MES      PaymentParcelStartEnum = "PROXIMO_MES"
	PaymentParcelStartEnumPROXIMA_QUINZENA PaymentParcelStartEnum = "PROXIMA_QUINZENA"
)

func (e *PaymentParcelStartEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PaymentParcelStartEnum(s)
	case string:
		*e = PaymentParcelStartEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for PaymentParcelStartEnum: %T", src)
	}
	return nil
}

func (e PaymentParcelStartEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── PriceFormationEnum ───────────────────────────────────────────────────────

type PriceFormationEnum string

const (
	PriceFormationEnumINFORMADO               PriceFormationEnum = "INFORMADO"
	PriceFormationEnumCUSTO_MEDIO             PriceFormationEnum = "CUSTO_MEDIO"
	PriceFormationEnumCUSTO_STANDARD_TOTAL    PriceFormationEnum = "CUSTO_STANDARD_TOTAL"
	PriceFormationEnumCUSTO_STANDARD_MATERIAL PriceFormationEnum = "CUSTO_STANDARD_MATERIAL"
	PriceFormationEnumINFORMADO_SEM_ICMS      PriceFormationEnum = "INFORMADO_SEM_ICMS"
	PriceFormationEnumMAT_OPER                PriceFormationEnum = "MAT_OPER"
	PriceFormationEnumTABELA_CUSTO            PriceFormationEnum = "TABELA_CUSTO"
	PriceFormationEnumTRANSFERENCIA_IPI       PriceFormationEnum = "TRANSFERENCIA_IPI"
	PriceFormationEnumTRANSFERENCIA_UF        PriceFormationEnum = "TRANSFERENCIA_UF"
)

func (e *PriceFormationEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PriceFormationEnum(s)
	case string:
		*e = PriceFormationEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for PriceFormationEnum: %T", src)
	}
	return nil
}

func (e PriceFormationEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── InvoiceTypeEnum ──────────────────────────────────────────────────────────

type InvoiceTypeEnum string

const (
	InvoiceTypeEnumVENDA                  InvoiceTypeEnum = "VENDA"
	InvoiceTypeEnumDEVOLUCAO              InvoiceTypeEnum = "DEVOLUCAO"
	InvoiceTypeEnumREMESSA                InvoiceTypeEnum = "REMESSA"
	InvoiceTypeEnumREMESSA_CONSIGNACAO    InvoiceTypeEnum = "REMESSA_CONSIGNACAO"
	InvoiceTypeEnumREMESSA_ARMAZENAGEM    InvoiceTypeEnum = "REMESSA_ARMAZENAGEM"
	InvoiceTypeEnumREMESSA_BENEFICIAMENTO InvoiceTypeEnum = "REMESSA_BENEFICIAMENTO"
	InvoiceTypeEnumRETORNO_BENEFICIAMENTO InvoiceTypeEnum = "RETORNO_BENEFICIAMENTO"
	InvoiceTypeEnumSIMPLES_REMESSA        InvoiceTypeEnum = "SIMPLES_REMESSA"
	InvoiceTypeEnumTRANSFERENCIA          InvoiceTypeEnum = "TRANSFERENCIA"
	InvoiceTypeEnumVENDA_CONSIGNACAO      InvoiceTypeEnum = "VENDA_CONSIGNACAO"
	InvoiceTypeEnumCOMPLEMENTAR_ICM       InvoiceTypeEnum = "COMPLEMENTAR_ICM"
	InvoiceTypeEnumCOMPLEMENTAR_IPI       InvoiceTypeEnum = "COMPLEMENTAR_IPI"
	InvoiceTypeEnumDEMONSTRACAO           InvoiceTypeEnum = "DEMONSTRACAO"
	InvoiceTypeEnumEMPRESTIMO             InvoiceTypeEnum = "EMPRESTIMO"
	InvoiceTypeEnumFATURAMENTO_ANTECIPADO InvoiceTypeEnum = "FATURAMENTO_ANTECIPADO"
	InvoiceTypeEnumPRESTACAO_SERVICOS     InvoiceTypeEnum = "PRESTACAO_SERVICOS"
	InvoiceTypeEnumOUTROS                 InvoiceTypeEnum = "OUTROS"
)

func (e *InvoiceTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = InvoiceTypeEnum(s)
	case string:
		*e = InvoiceTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for InvoiceTypeEnum: %T", src)
	}
	return nil
}

func (e InvoiceTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── InvoiceStockEnum ─────────────────────────────────────────────────────────

type InvoiceStockEnum string

const (
	InvoiceStockEnumATUALIZA              InvoiceStockEnum = "ATUALIZA"
	InvoiceStockEnumNAO_ATUALIZA          InvoiceStockEnum = "NAO_ATUALIZA"
	InvoiceStockEnumTRANSFERENCIA_EXTERNA InvoiceStockEnum = "TRANSFERENCIA_EXTERNA"
)

func (e *InvoiceStockEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = InvoiceStockEnum(s)
	case string:
		*e = InvoiceStockEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for InvoiceStockEnum: %T", src)
	}
	return nil
}

func (e InvoiceStockEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── InvoiceICMSTypeEnum ──────────────────────────────────────────────────────

type InvoiceICMSTypeEnum string

const (
	InvoiceICMSTypeEnumTRIBUTADO InvoiceICMSTypeEnum = "TRIBUTADO"
	InvoiceICMSTypeEnumISENTO    InvoiceICMSTypeEnum = "ISENTO"
	InvoiceICMSTypeEnumOUTROS    InvoiceICMSTypeEnum = "OUTROS"
)

func (e *InvoiceICMSTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = InvoiceICMSTypeEnum(s)
	case string:
		*e = InvoiceICMSTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for InvoiceICMSTypeEnum: %T", src)
	}
	return nil
}

func (e InvoiceICMSTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── DocumentTypeEnum ─────────────────────────────────────────────────────────

type DocumentTypeEnum string

const (
	DocumentTypeEnumCNPJ        DocumentTypeEnum = "CNPJ"
	DocumentTypeEnumCPF         DocumentTypeEnum = "CPF"
	DocumentTypeEnumESTRANGEIRO DocumentTypeEnum = "ESTRANGEIRO"
	DocumentTypeEnumISENTO      DocumentTypeEnum = "ISENTO"
)

func (e *DocumentTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = DocumentTypeEnum(s)
	case string:
		*e = DocumentTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for DocumentTypeEnum: %T", src)
	}
	return nil
}

func (e DocumentTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── CustomerAddressTypeEnum ──────────────────────────────────────────────────

type CustomerAddressTypeEnum string

const (
	CustomerAddressTypeEnumCOBRANCA  CustomerAddressTypeEnum = "COBRANCA"
	CustomerAddressTypeEnumENTREGA   CustomerAddressTypeEnum = "ENTREGA"
	CustomerAddressTypeEnumCOMERCIAL CustomerAddressTypeEnum = "COMERCIAL"
	CustomerAddressTypeEnumOUTRO     CustomerAddressTypeEnum = "OUTRO"
)

func (e *CustomerAddressTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CustomerAddressTypeEnum(s)
	case string:
		*e = CustomerAddressTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for CustomerAddressTypeEnum: %T", src)
	}
	return nil
}

func (e CustomerAddressTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── PaymentCondVisibilityEnum ────────────────────────────────────────────────

type PaymentCondVisibilityEnum string

const (
	PaymentCondVisibilityEnumSOMENTE_VINCULADOS  PaymentCondVisibilityEnum = "SOMENTE_VINCULADOS"
	PaymentCondVisibilityEnumVINCULADOS_E_NENHUM PaymentCondVisibilityEnum = "VINCULADOS_E_NENHUM"
	PaymentCondVisibilityEnumTODOS               PaymentCondVisibilityEnum = "TODOS"
)

func (e *PaymentCondVisibilityEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PaymentCondVisibilityEnum(s)
	case string:
		*e = PaymentCondVisibilityEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for PaymentCondVisibilityEnum: %T", src)
	}
	return nil
}

func (e PaymentCondVisibilityEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── TableCompositionEnum ─────────────────────────────────────────────────────

type TableCompositionEnum string

const (
	TableCompositionEnumEXWORK TableCompositionEnum = "EXWORK"
	TableCompositionEnumCIF    TableCompositionEnum = "CIF"
	TableCompositionEnumFOB    TableCompositionEnum = "FOB"
)

func (e *TableCompositionEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TableCompositionEnum(s)
	case string:
		*e = TableCompositionEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for TableCompositionEnum: %T", src)
	}
	return nil
}

func (e TableCompositionEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── TableTypeEnum ────────────────────────────────────────────────────────────

type TableTypeEnum string

const (
	TableTypeEnumNORMAL      TableTypeEnum = "NORMAL"
	TableTypeEnumPROMOCIONAL TableTypeEnum = "PROMOCIONAL"
)

func (e *TableTypeEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TableTypeEnum(s)
	case string:
		*e = TableTypeEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for TableTypeEnum: %T", src)
	}
	return nil
}

func (e TableTypeEnum) Value() (driver.Value, error) {
	return string(e), nil
}

// ─── BaseDateEnum ─────────────────────────────────────────────────────────────

type BaseDateEnum string

const (
	BaseDateEnumPEDIDO    BaseDateEnum = "PEDIDO"
	BaseDateEnumDATAATUAL BaseDateEnum = "DATA_ATUAL"
)

func (e *BaseDateEnum) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = BaseDateEnum(s)
	case string:
		*e = BaseDateEnum(s)
	default:
		return fmt.Errorf("unsupported scan type for BaseDateEnum: %T", src)
	}
	return nil
}

func (e BaseDateEnum) Value() (driver.Value, error) {
	return string(e), nil
}
