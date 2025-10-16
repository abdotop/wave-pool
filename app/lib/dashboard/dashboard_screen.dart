import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:wave_pool/dashboard/models/payment_model.dart';
import 'package.wave_pool/dashboard/services/payment_service.dart';

class DashboardScreen extends HookWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    // QR Scanner state
    final scannedValue = useState<String?>(null);
    final isScanning = useState<bool>(true);

    // Payments state
    final paymentService = useMemoized(() => PaymentService());
    final paymentsFuture = useMemoized(() => paymentService.getPayments(), []);
    final paymentsSnapshot = useFuture(paymentsFuture);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Dashboard'),
      ),
      body: SingleChildScrollView(
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // --- QR Scanner Section ---
              if (isScanning.value)
                Container(
                  margin: const EdgeInsets.symmetric(horizontal: 20),
                  decoration: BoxDecoration(
                    border: Border.all(color: Colors.blueAccent, width: 3),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  height: 300,
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(16),
                    child: MobileScanner(
                      onDetect: (capture) {
                        final List<Barcode> barcodes = capture.barcodes;
                        if (barcodes.isNotEmpty) {
                          isScanning.value = false;
                          scannedValue.value = barcodes.first.rawValue ?? "QR invalide";
                        }
                      },
                    ),
                  ),
                )
              else
                Container(
                  height: 300,
                  alignment: Alignment.center,
                  margin: const EdgeInsets.symmetric(horizontal: 20),
                  decoration: BoxDecoration(
                    border: Border.all(color: Colors.grey.shade300, width: 2),
                    borderRadius: BorderRadius.circular(16),
                    color: Colors.grey.shade100,
                  ),
                  child: const Icon(
                    Icons.qr_code_2,
                    size: 100,
                    color: Colors.grey,
                  ),
                ),
              const SizedBox(height: 20),
              Text(
                scannedValue.value == null
                    ? "Aucun code scanné"
                    : "✅ Code détecté : ${scannedValue.value}",
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
              ),
              const SizedBox(height: 10),
              if (!isScanning.value)
                ElevatedButton.icon(
                  onPressed: () {
                    scannedValue.value = null;
                    isScanning.value = true;
                  },
                  icon: const Icon(Icons.refresh),
                  label: const Text("Scanner à nouveau"),
                ),

              // --- Divider ---
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 20.0),
                child: Divider(thickness: 2),
              ),

              // --- Transactions Section ---
              const Text(
                'Transactions Récentes',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 10),
              if (paymentsSnapshot.connectionState == ConnectionState.waiting)
                const Center(child: CircularProgressIndicator())
              else if (paymentsSnapshot.hasError)
                Center(child: Text('Error: ${paymentsSnapshot.error}'))
              else if (!paymentsSnapshot.hasData || paymentsSnapshot.data!.isEmpty)
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