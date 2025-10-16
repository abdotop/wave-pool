class Payment {
  final String id;
  final double amount;
  final String currency;
  final String merchant;
  final DateTime date;
  final PaymentStatus status;

  Payment({
    required this.id,
    required this.amount,
    required this.currency,
    required this.merchant,
    required this.date,
    required this.status,
  });
}

enum PaymentStatus {
  completed,
  pending,
  failed,
}