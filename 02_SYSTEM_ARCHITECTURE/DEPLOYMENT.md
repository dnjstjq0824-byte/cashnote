# 컨테이너 및 프로덕션 배포 아키텍처 (Deployment Architecture)

**문서 ID**: ARCH-DEPLOY-004  

---

## 1. 환경 구성 (Environments)

| 환경 명칭 | 용도 | 외부 금융/공공 API 연결 정책 | 데이터베이스 구성 |
|---|---|---|---|
| `local-dev` | 개발자/에이전트 로컬 Docker Compose | 100% 로컬 WireMock / Sandbox Stub 사용 | PostgreSQL 16 단일 컨테이너 + 시드 데이터 |
| `ci-test` | GitHub Actions / GitLab CI 자동 검증 | Ephemeral Testcontainers + 계약 테스트 픽스처 | 매 테스트 실행마다 초기화되는 격리 컨테이너 |
| `staging` | 통합 QA 및 파트너 연동 검증 | 공공데이터포털 개발키 / PG·팝빌 샌드박스 API 연결 | Multi-AZ RDS/PostgreSQL (익명화 데이터만 허용) |
| `production` | 실제 소상공인 상용 서비스 | `provider_catalog.is_live_enabled=true` 승인 기관만 연결 (전용 고정 Egress IP + mTLS) | PostgreSQL 16 Primary + Sync Standby + Read Replica + PITR |

---

## 2. 표준 Docker Compose 토폴로지 (모듈형 모놀리스 + 워커)

프로덕션 및 스테이징 표준 컨테이너 구성은 다음과 같다:
- `bn-mos-gateway`: Envoy Proxy (TLS 1.3, Rate Limit, CORS, Security Headers)
- `bn-mos-web-merchant`: Nuxt 3 SSR/Hybrid 컨테이너 (Port 3000)
- `bn-mos-web-admin`: Vue 3 Admin 정적/Nginx 컨테이너 (내부 관리망 전용, Port 3001)
- `bn-mos-api`: Go Core API Server (Port 8080, Healthcheck `/v1/health/ready`)
- `bn-mos-worker-sync`: Go External Connector & File Parser Worker
- `bn-mos-worker-recon`: Go Settlement Reconciliation & Notification Worker
- `bn-mos-postgres`: PostgreSQL 16 (`pg_hba.conf` TLS 강제, PgBouncer 연동)
- `bn-mos-redis`: Redis 7.2 (AOF 활성화, ACL 패스워드 보호)
- `bn-mos-opensearch`: OpenSearch 2.15 (`security` 플러그인 활성화, `analysis-nori` 탑재)
- `bn-mos-otel-collector`: OpenTelemetry Collector (메트릭·로그·트레이스 수집 및 PII 스크러빙)
