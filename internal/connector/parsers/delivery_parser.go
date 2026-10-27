package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"merchant-os/internal/platform/envelope"
	apperr "merchant-os/internal/platform/errors"
)

// DeliveryPlatform represents the food delivery service
type DeliveryPlatform string

const (
	PlatformBaemin      DeliveryPlatform = "BAEMIN"       // 배달의민족
	PlatformCoupangEats DeliveryPlatform = "COUPANG_EATS" // 쿠팡이츠
	PlatformYogiyo      DeliveryPlatform = "YOGIYO"       // 요기요
)

// DeliverySettlementRecord represents parsed 11 core financial attributes
type DeliverySettlementRecord struct {
	ID                     uuid.UUID                `json:"id"`
	StoreID                uuid.UUID                `json:"store_id"`
	Platform               DeliveryPlatform         `json:"platform"`
	SettlementDate         time.Time                `json:"settlement_date"`
	GrossOrderAmount       decimal.Decimal          `json:"gross_order_amount"`       // 주문금액 합계
	CanceledOrderAmount    decimal.Decimal          `json:"canceled_order_amount"`    // 취소금액
	PlatformMediationFee   decimal.Decimal          `json:"platform_mediation_fee"`   // 중개수수료
	DeliveryAgencyFee      decimal.Decimal          `json:"delivery_agency_fee"`      // 배달비(사장님부담)
	PaymentGatewayFee      decimal.Decimal          `json:"payment_gateway_fee"`      // 결제수수료
	MerchantCouponCost     decimal.Decimal          `json:"merchant_coupon_cost"`     // 사장님부담 할인쿠폰
	AdProductDeduction     decimal.Decimal          `json:"ad_product_deduction"`     // 광고비 차감
	VatOnFees              decimal.Decimal          `json:"vat_on_fees"`              // 수수료 부가세
	NetSettledAmount       decimal.Decimal          `json:"net_settled_amount"`       // 실 입금 예정액
	SourceLevel            envelope.ProvenanceLevel `json:"source_level"`             // LEVEL_2_FILE
	BankReconciled         bool                     `json:"bank_reconciled"`
}

// ParseDeliverySettlementCSV parses official settlement CSV files downloaded by merchants
func ParseDeliverySettlementCSV(storeID uuid.UUID, platform DeliveryPlatform, r io.Reader) ([]DeliverySettlementRecord, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, apperr.NewBadRequest("정산 파일 형식이 올바르지 않습니다.")
	}

	headerMap := make(map[string]int)
	for idx, h := range headers {
		clean := strings.TrimSpace(h)
		headerMap[clean] = idx
	}

	var records []DeliverySettlementRecord
	rowNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.NewBadRequest(fmt.Sprintf("%d번째 행 파싱 실패: %v", rowNum, err))
		}
		rowNum++

		if len(row) < 5 {
			continue // skip empty or trailing lines
		}

		settleDate, _ := time.Parse("2006-01-02", strings.TrimSpace(row[0]))
		if settleDate.IsZero() {
			settleDate = time.Now()
		}

		gross, _ := decimal.NewFromString(strings.ReplaceAll(row[1], ",", ""))
		canceled, _ := decimal.NewFromString(strings.ReplaceAll(row[2], ",", ""))
		medFee, _ := decimal.NewFromString(strings.ReplaceAll(row[3], ",", ""))
		delivFee, _ := decimal.NewFromString(strings.ReplaceAll(row[4], ",", ""))
		pgFee, _ := decimal.NewFromString(strings.ReplaceAll(row[5], ",", ""))
		coupon, _ := decimal.NewFromString(strings.ReplaceAll(row[6], ",", ""))
		adFee, _ := decimal.NewFromString(strings.ReplaceAll(row[7], ",", ""))
		vatFee, _ := decimal.NewFromString(strings.ReplaceAll(row[8], ",", ""))

		// Net Settle = (Gross - Canceled) - (MedFee + DelivFee + PGFee + Coupon + AdFee + VatFee)
		totalDeductions := medFee.Add(delivFee).Add(pgFee).Add(coupon).Add(adFee).Add(vatFee)
		netSettled := gross.Sub(canceled).Sub(totalDeductions)

		rec := DeliverySettlementRecord{
			ID:                   uuid.New(),
			StoreID:              storeID,
			Platform:             platform,
			SettlementDate:       settleDate,
			GrossOrderAmount:     gross,
			CanceledOrderAmount:  canceled,
			PlatformMediationFee: medFee,
			DeliveryAgencyFee:    delivFee,
			PaymentGatewayFee:    pgFee,
			MerchantCouponCost:   coupon,
			AdProductDeduction:   adFee,
			VatOnFees:            vatFee,
			NetSettledAmount:     netSettled,
			SourceLevel:          envelope.Level2File,
			BankReconciled:       false,
		}

		records = append(records, rec)
	}

	if len(records) == 0 {
		return nil, apperr.NewBadRequest("유효한 정산 데이터 행이 없습니다.")
	}

	return records, nil
}
