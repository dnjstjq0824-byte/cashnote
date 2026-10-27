# 외부 데이터 커넥터 표준 계약 명세 (Connector Contract Specification)

**문서 ID**: API-CONN-003  

---

## 1. Go `ConnectorProvider` 표준 인터페이스

모든 외부 기관(여신협회, 오픈뱅킹, 홈택스 ASP, PG, VAN, POS, 배달앱, 리뷰 플랫폼) 어댑터는 다음 Go 인터페이스를 구현해야 한다:

```go
package connector

import (
    "context"
    "io"
    "time"
)

type AccessStatus string

const (
    StatusPublicAPI          AccessStatus = "PUBLIC_API"
    StatusApprovalRequired   AccessStatus = "APPROVAL_REQUIRED"
    StatusContractRequired   AccessStatus = "CONTRACT_REQUIRED"
    StatusLicenseRequired    AccessStatus = "LICENSE_REQUIRED"
    StatusPartnerOnly        AccessStatus = "PARTNER_ONLY"
    StatusCommercialProvider AccessStatus = "COMMERCIAL_PROVIDER"
    StatusBlocked            AccessStatus = "BLOCKED"
)

type SyncWindow struct {
    StoreID      string
    ConnectionID string
    From         time.Time
    To           time.Time
    Cursor       string
}

type ProviderCapability struct {
    ProviderCode       string
    AccessStatus       AccessStatus
    SupportsLevel2File bool
    SupportsLevel3API  bool
    IsLiveEnabled      bool
}

type ConnectorProvider interface {
    Capabilities() ProviderCapability
    // ValidateLiveEligibility returns ErrConnectorBlocked if contract/regulatory approval is not active.
    ValidateLiveEligibility(ctx context.Context, storeID string) error
    // FetchAuthorizedAPI executes Level 3 API sync only when ValidateLiveEligibility passes.
    FetchAuthorizedAPI(ctx context.Context, win SyncWindow) (*NormalizedBatch, error)
    // ParseImportFile executes Level 2 official CSV/Excel parsing without requiring external API credentials.
    ParseImportFile(ctx context.Context, storeID string, fileReader io.Reader, fileName string) (*NormalizedBatch, error)
}
```
