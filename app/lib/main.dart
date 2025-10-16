import 'package:flutter/material.dart';
import 'package:flutter_native_splash/flutter_native_splash.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:flutter_hooks/flutter_hooks.dart';


void main() {
  WidgetsBinding widgetsBinding = WidgetsFlutterBinding.ensureInitialized();
  FlutterNativeSplash.preserve(widgetsBinding: widgetsBinding);
  runApp(const App());
}

class App extends HookWidget {
  const App({super.key});


  @override
  Widget build(BuildContext context) {
    FlutterNativeSplash.remove();
    return MaterialApp(
      title: 'Wave Pool',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.lightBlueAccent),
      ),
      home: QrScannerPage(),
    );
  }
}



class QrScannerPage extends HookWidget {
  const QrScannerPage({super.key});

  @override
  Widget build(BuildContext context) {
    final scannedValue = useState<String?>(null);
    final isScanning = useState<bool>(true);

    return Scaffold(
      body: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 20),
            child: Image.asset(
              'assets/icon/icon.png',
              height: 120,
            ),
          ),

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
                      scannedValue.value =
                          barcodes.first.rawValue ?? "QR invalide";
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

          const SizedBox(height: 30),

          Text(
            scannedValue.value == null
                ? "Aucun code scanné"
                : "✅ Code détecté : ${scannedValue.value}",
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
          ),

          const SizedBox(height: 20),

          if (!isScanning.value)
            ElevatedButton.icon(
              onPressed: () {
                scannedValue.value = null;
                isScanning.value = true;
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: Colors.blueAccent,
                padding:
                    const EdgeInsets.symmetric(horizontal: 32, vertical: 12),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              icon: const Icon(Icons.refresh, color: Colors.white),
              label: const Text(
                "Scanner à nouveau",
                style: TextStyle(color: Colors.white),
              ),
            ),
        ],
      ),
    );
  }
}