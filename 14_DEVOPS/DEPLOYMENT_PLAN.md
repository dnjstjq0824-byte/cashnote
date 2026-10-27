# 무중단 프로덕션 배포 및 환경 분리 계획서 (Deployment Plan)

**문서 ID**: DEVOPS-DEPLOY-002  

---

## 1. 무중단 데이터베이스 마이그레이션 (Expand-and-Contract Pattern)

실시간 POS 웹훅과 사장님 장부 조회가 24시간 일어나는 상용 환경에서 테이블 락(Table Lock)으로 인한 장애를 막기 위해 다음 원칙을 강제한다:
1. **인덱스 생성**: 반드시 `CREATE INDEX CONCURRENTLY`를 사용하여 읽기/쓰기 차단을 방지한다.
2. **컬럼 변경 3단계 분리**:
   - *Release 1 (Expand)*: 신규 `NULL` 허용 컬럼 추가 및 애플리케이션 Dual-Write 배포.
   - *Release 2 (Backfill)*: 배치 워커가 1,000행 단위 청크로 과거 데이터 백필 및 `NOT NULL` CHECK(`NOT VALID` → `VALIDATE CONSTRAINT`) 적용.
   - *Release 3 (Contract)*: 구 컬럼 참조 코드 제거 후 컬럼 드롭.
