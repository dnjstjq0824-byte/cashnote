import 'package:flutter/material.dart';
import 'core/theme/colors.dart';
import 'core/provenance/data_provenance_badge.dart';

void main() {
  runApp(const BNMerchantApp());
}

class BNMerchantApp extends StatelessWidget {
  const BNMerchantApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'BN Merchant OS',
      theme: ThemeData(
        fontFamily: 'Pretendard',
        scaffoldBackgroundColor: BNColors.bgSurface,
        useMaterial3: true,
      ),
      home: const MerchantHomeScreen(),
    );
  }
}

class MerchantHomeScreen extends StatelessWidget {
  const MerchantHomeScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: BNColors.brandPrimary,
        elevation: 0,
        title: const Text(
          'BN Merchant OS',
          style: TextStyle(
            color: Colors.white,
            fontSize: 18,
            fontWeight: FontWeight.bold,
          ),
        ),
        actions: const [
          Padding(
            padding: EdgeInsets.only(right: 16),
            child: Center(
              child: DataProvenanceBadge(
                level: ProvenanceLevel.level4Reconciled,
              ),
            ),
          )
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Today Net Revenue Hero Card
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: BNColors.borderSubtle),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withOpacity(0.04),
                    blurRadius: 10,
                    offset: const Offset(0, 4),
                  )
                ],
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    '오늘 입금 완료액 (대사 완료)',
                    style: TextStyle(
                      fontSize: 13,
                      color: BNColors.textMuted,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    '1,452,000 원',
                    style: TextStyle(
                      fontSize: 28,
                      fontWeight: FontWeight.bold,
                      color: BNColors.financeReconciled,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      _buildSubMetric('승인액', '1,500,000원', BNColors.financeAuthorized),
                      const SizedBox(width: 16),
                      _buildSubMetric('수수료', '-48,000원', BNColors.financeExpense),
                    ],
                  )
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Level 2 File Parser CTA
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: const Color(0xFFEFF6FF),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFBFDBFE)),
              ),
              child: Row(
                children: [
                  const Icon(Icons.upload_file, color: BNColors.financeAuthorized),
                  const SizedBox(width: 12),
                  const Expanded(
                    child: Text(
                      '배달/카드 정산서 엑셀 파일을 업로드하여 누락금을 진단하세요 (Level 2).',
                      style: TextStyle(
                        fontSize: 12,
                        color: Color(0xFF1E3A8A),
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
                ],
              ),
            )
          ],
        ),
      ),
    );
  }

  Widget _buildSubMetric(String label, String value, Color color) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 11, color: BNColors.textMuted),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, color: color),
        ),
      ],
    );
  }
}

