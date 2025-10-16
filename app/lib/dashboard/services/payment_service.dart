import 'package:wave_pool/dashboard/models/payment_model.dart';

class PaymentService {
  Future<List<Payment>> getPayments() async {
    // Simulate a network delay
    await Future.delayed(const Duration(seconds: 1));

    // Return a list of mock payments
    return [
      Payment(
        id: '1',
        amount: 1500.0,
        currency: 'XOF',
        merchant: 'Boutique Chez Ali',
        date: DateTime.now().subtract(const Duration(days: 1)),
        status: PaymentStatus.completed,
      ),
      Payment(
        id: '2',
        amount: 250.0,
        currency: 'XOF',
        merchant: 'Orange SN',
        date: DateTime.now().subtract(const Duration(days: 2)),
        status: PaymentStatus.completed,
      ),
      Payment(
        id: '3',
        amount: 5000.0,
        currency: 'XOF',
        merchant: 'Supermarché Prix-Bas',
        date: DateTime.now().subtract(const Duration(days: 2)),
        status: PaymentStatus.pending,
      ),
      Payment(
        id: '4',
        amount: 750.0,
        currency: 'XOF',
        merchant: 'Pharmacie La Confiance',
        date: DateTime.now().subtract(const Duration(days: 3)),
        status: PaymentStatus.failed,
      ),
    ];
  }
}