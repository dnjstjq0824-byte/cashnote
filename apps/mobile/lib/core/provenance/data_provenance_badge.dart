import 'package:flutter/material.dart';
import '../theme/colors.dart';

enum ProvenanceLevel {
  level1Manual,
  level2File,
  level3Api,
  level4Reconciled,
}

class DataProvenanceBadge extends StatelessWidget {
  final ProvenanceLevel level;
  final String? updatedAt;

  const DataProvenanceBadge({
    Key? key,
    required this.level,
    this.updatedAt,
  }) : super(key: key);

  String get label {
    switch (level) {
      case ProvenanceLevel.level4Reconciled:
        return '은행 대사 완료 (L4)';
      case ProvenanceLevel.level3Api:
        return '공식 API (L3)';
      case ProvenanceLevel.level2File:
        return '정산서 파일 (L2)';
      case ProvenanceLevel.level1Manual:
      default:
        return '수기 입력 (L1)';
    }
  }

  Color get bgColor {
    switch (level) {
      case ProvenanceLevel.level4Reconciled:
        return const Color(0xFFECFDF5);
      case ProvenanceLevel.level3Api:
        return const Color(0xFFF5F3FF);
      case ProvenanceLevel.level2File:
        return const Color(0xFFEFF6FF);
      case ProvenanceLevel.level1Manual:
      default:
        return const Color(0xFFF1F5F9);
    }
  }

  Color get textColor {
    switch (level) {
      case ProvenanceLevel.level4Reconciled:
        return BNColors.financeReconciled;
      case ProvenanceLevel.level3Api:
        return const Color(0xFF7C3AED);
      case ProvenanceLevel.level2File:
        return BNColors.financeAuthorized;
      case ProvenanceLevel.level1Manual:
      default:
        return BNColors.textMuted;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(20),
        border: Border.parse('1px solid ${textColor.withOpacity(0.3)}'),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: textColor,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: TextStyle(
              color: textColor,
              fontSize: 11,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

