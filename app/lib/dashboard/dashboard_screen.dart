import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:wave_pool/auth/auth_service.dart';
import 'package:wave_pool/core/secure_storage_service.dart';
import 'package:wave_pool/dashboard/models/payment_model.dart';
import 'package:wave_pool/dashboard/services/payment_service.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

class DashboardScreen extends HookWidget {
  final AuthService? authService;
  const DashboardScreen({super.key, this.authService});

  @override
  Widget build(BuildContext context) {
    final scannedValue = useState<String?>(null);
    final open = useState<bool>(false);
    final paymentService = useMemoized(() => PaymentService());
    final paymentsFuture = useMemoized(() => paymentService.getPayments(), []);
    final paymentsSnapshot = useFuture(paymentsFuture);
    final authService = useMemoized(() {
      if (this.authService != null) {
        return this.authService!;
      }
      final secureStorage = SecureStorageService();
      final authSvc = AuthService(secureStorage, http.Client());
      return authSvc;
    }, [this.authService]);
    return Scaffold(
      appBar: AppBar(
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () {
              authService.logout();
              if (context.mounted) {
                context.go('/auth');
              }
            },
          ),
        ],
      ),
      body: SingleChildScrollView(
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (open.value)
                Container(
                  margin: const EdgeInsets.symmetric(horizontal: 20),
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(16),
                  ),
                  height: 170,
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(16),
                    child: MobileScanner(
                      onDetect: (capture) {
                        final List<Barcode> barcodes = capture.barcodes;
                        if (barcodes.isNotEmpty) {
                          open.value = false;
                          scannedValue.value =
                              barcodes.first.rawValue ?? "QR invalide";
                        }
                      },
                    ),
                  ),
                )
              else
                GestureDetector(
                  onTap: () {
                    scannedValue.value = null;
                    open.value = true;
                  },
                  child: Container(
                    height: 170,
                    alignment: Alignment.center,
                    margin: const EdgeInsets.symmetric(horizontal: 20),
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(16),
                      color: Colors.lightBlueAccent.shade100,
                    ),
                    child: Image.asset(
                      'assets/icon/qr_card.png',
                      fit: BoxFit.cover,
                    ),
                  ),
                ),
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 20.0),
                child: Divider(thickness: 2),
              ),

              const Text(
                'Transactions Récentes',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 10),
              if (paymentsSnapshot.connectionState == ConnectionState.waiting)
                const Center(child: CircularProgressIndicator())
              else if (paymentsSnapshot.hasError)
                Center(child: Text('Error: ${paymentsSnapshot.error}'))
              else if (!paymentsSnapshot.hasData ||
                  paymentsSnapshot.data!.isEmpty)
                const Center(child: Text('Aucune transaction trouvée.'))
              else
                ListView.builder(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  itemCount: paymentsSnapshot.data!.length,
                  itemBuilder: (context, index) {
                    final payment = paymentsSnapshot.data![index];
                    return ListTile(
                      leading: Icon(
                        payment.status == PaymentStatus.completed
                            ? Icons.check_circle
                            : payment.status == PaymentStatus.pending
                            ? Icons.hourglass_empty
                            : Icons.cancel,
                        color: payment.status == PaymentStatus.completed
                            ? Colors.green
                            : payment.status == PaymentStatus.pending
                            ? Colors.orange
                            : Colors.red,
                      ),
                      title: Text(payment.merchant),
                      subtitle: Text('${payment.amount} ${payment.currency}'),
                      trailing: Text(
                        '${payment.date.day}/${payment.date.month}/${payment.date.year}',
                      ),
                    );
                  },
                ),
            ],
          ),
        ),
      ),
    );
  }
}
