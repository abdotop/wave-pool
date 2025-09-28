import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:http/http.dart' as http;
import 'dart:convert';

void main() {
  runApp(const WavePoolApp());
}

class WavePoolApp extends StatelessWidget {
  const WavePoolApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Wave Pool',
      theme: ThemeData(
        primarySwatch: Colors.blue,
        useMaterial3: true,
      ),
      home: const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _selectedIndex = 0;

  final List<Widget> _pages = [
    const PaymentSimulatorPage(),
    const QRScannerPage(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Wave Pool'),
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
      ),
      body: _pages[_selectedIndex],
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _selectedIndex,
        onTap: (index) => setState(() => _selectedIndex = index),
        items: const [
          BottomNavigationBarItem(
            icon: Icon(Icons.payment),
            label: 'Payment',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.qr_code_scanner),
            label: 'Scan QR',
          ),
        ],
      ),
    );
  }
}

class PaymentSimulatorPage extends StatefulWidget {
  const PaymentSimulatorPage({super.key});

  @override
  State<PaymentSimulatorPage> createState() => _PaymentSimulatorPageState();
}

class _PaymentSimulatorPageState extends State<PaymentSimulatorPage> {
  String? sessionId;
  Map<String, dynamic>? sessionData;
  bool isLoading = false;
  String? errorMessage;

  @override
  void initState() {
    super.initState();
    // Check if the app was opened with a deep link
    _handleInitialLink();
  }

  Future<void> _handleInitialLink() async {
    // In a real Flutter app, this would use app_links or similar package
    // For demonstration purposes, we'll show how it would work
    
    // Example: wavepool://pay/cos-12345
    // This would be parsed from the initial intent/URL
    
    // For now, we'll simulate this with a demo session ID
    // In production, this would come from the app launch intent
    _handleDeepLink('cos-demo123');
  }

  Future<void> _handleDeepLink(String sessionId) async {
    setState(() {
      this.sessionId = sessionId;
      isLoading = true;
      errorMessage = null;
    });

    try {
      // Fetch session details from the backend
      final response = await http.get(
        Uri.parse('http://localhost:8081/v1/checkout/sessions/$sessionId'),
        headers: {
          'Authorization': 'Bearer demo_api_key', // In real app, this would be configurable
        },
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          sessionData = data;
          isLoading = false;
        });
      } else {
        setState(() {
          errorMessage = 'Failed to load payment session';
          isLoading = false;
        });
      }
    } catch (e) {
      setState(() {
        errorMessage = 'Error: \$e';
        isLoading = false;
      });
    }
  }

  Future<void> _simulatePayment(String status) async {
    if (sessionId == null) return;

    setState(() => isLoading = true);

    try {
      final response = await http.post(
        Uri.parse('http://localhost:8081/pay/$sessionId'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'status': status}),
      );

      if (response.statusCode == 200) {
        _showResultDialog(status == 'succeeded' ? 'Payment Successful!' : 'Payment Failed!');
      } else {
        _showResultDialog('Error processing payment');
      }
    } catch (e) {
      _showResultDialog('Network error: \$e');
    }

    setState(() => isLoading = false);
  }

  void _showResultDialog(String message) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Payment Result'),
        content: Text(message),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.of(context).pop();
              setState(() {
                sessionId = null;
                sessionData = null;
              });
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (sessionId == null) {
      return const Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.qr_code_scanner, size: 64, color: Colors.grey),
            SizedBox(height: 16),
            Text(
              'Scan a QR code or open a payment link',
              style: TextStyle(fontSize: 18, color: Colors.grey),
            ),
            SizedBox(height: 8),
            Text(
              'Use the QR Scanner tab to scan payment codes',
              style: TextStyle(fontSize: 14, color: Colors.grey),
            ),
          ],
        ),
      );
    }

    if (isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (errorMessage != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error, size: 64, color: Colors.red),
            const SizedBox(height: 16),
            Text(errorMessage!, style: const TextStyle(color: Colors.red)),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () => setState(() {
                sessionId = null;
                errorMessage = null;
              }),
              child: const Text('Back'),
            ),
          ],
        ),
      );
    }

    if (sessionData == null) {
      return const Center(child: CircularProgressIndicator());
    }

    return Padding(
      padding: const EdgeInsets.all(16.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text(
                    'Payment Details',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 16),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text('Amount:', style: TextStyle(fontSize: 16)),
                      Text(
                        '\${sessionData!['amount']} \${sessionData!['currency']}',
                        style: const TextStyle(
                          fontSize: 24,
                          fontWeight: FontWeight.bold,
                          color: Colors.blue,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text('Merchant:', style: TextStyle(fontSize: 16)),
                      Text(
                        sessionData!['business_name'] ?? 'Unknown',
                        style: const TextStyle(fontSize: 16),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text('Session ID:', style: TextStyle(fontSize: 12, color: Colors.grey)),
                      Text(
                        sessionId!,
                        style: const TextStyle(fontSize: 12, color: Colors.grey, fontFamily: 'monospace'),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            'Simulate Payment Result:',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          ElevatedButton.icon(
            onPressed: isLoading ? null : () => _simulatePayment('succeeded'),
            icon: const Icon(Icons.check_circle, color: Colors.white),
            label: const Text('Simulate Success', style: TextStyle(fontSize: 18)),
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.green,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 16),
            ),
          ),
          const SizedBox(height: 12),
          ElevatedButton.icon(
            onPressed: isLoading ? null : () => _simulatePayment('failed'),
            icon: const Icon(Icons.error, color: Colors.white),
            label: const Text('Simulate Failure', style: TextStyle(fontSize: 18)),
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.red,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 16),
            ),
          ),
          const SizedBox(height: 24),
          TextButton(
            onPressed: () => setState(() {
              sessionId = null;
              sessionData = null;
            }),
            child: const Text('Cancel / Back'),
          ),
        ],
      ),
    );
  }
}

class QRScannerPage extends StatefulWidget {
  const QRScannerPage({super.key});

  @override
  State<QRScannerPage> createState() => _QRScannerPageState();
}

class _QRScannerPageState extends State<QRScannerPage> {
  MobileScannerController cameraController = MobileScannerController();
  bool isScanning = false;

  @override
  void dispose() {
    cameraController.dispose();
    super.dispose();
  }

  void _onDetect(BarcodeCapture capture) {
    if (isScanning) return;

    final List<Barcode> barcodes = capture.barcodes;
    for (final barcode in barcodes) {
      final String? code = barcode.rawValue;
      if (code != null && code.startsWith('wavepool://pay/')) {
        setState(() => isScanning = true);
        _handleQRCode(code);
        break;
      }
    }
  }

  void _handleQRCode(String url) {
    // Parse the deep link URL
    // Expected format: wavepool://pay/session_id
    final uri = Uri.parse(url);
    if (uri.scheme == 'wavepool' && uri.pathSegments.length >= 2 && uri.pathSegments[0] == 'pay') {
      final sessionId = uri.pathSegments[1];
      
      // Navigate to payment simulator with session ID
      Navigator.push(
        context,
        MaterialPageRoute(
          builder: (context) => PaymentSimulatorPageWithSession(sessionId: sessionId),
        ),
      ).then((_) => setState(() => isScanning = false));
    } else {
      _showError('Invalid QR code. Expected Wave Pool payment link.');
    }
  }

  void _showError(String message) {
    setState(() => isScanning = false);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Expanded(
          child: MobileScanner(
            controller: cameraController,
            onDetect: _onDetect,
          ),
        ),
        Container(
          padding: const EdgeInsets.all(16),
          child: const Column(
            children: [
              Text(
                'Scan QR Code',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
              ),
              SizedBox(height: 8),
              Text(
                'Point your camera at a Wave Pool payment QR code',
                style: TextStyle(color: Colors.grey),
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class PaymentSimulatorPageWithSession extends StatefulWidget {
  final String sessionId;

  const PaymentSimulatorPageWithSession({super.key, required this.sessionId});

  @override
  State<PaymentSimulatorPageWithSession> createState() => _PaymentSimulatorPageWithSessionState();
}

class _PaymentSimulatorPageWithSessionState extends State<PaymentSimulatorPageWithSession> {
  Map<String, dynamic>? sessionData;
  bool isLoading = true;
  String? errorMessage;

  @override
  void initState() {
    super.initState();
    _loadSession();
  }

  Future<void> _loadSession() async {
    try {
      final response = await http.get(
        Uri.parse('http://localhost:8081/v1/checkout/sessions/\${widget.sessionId}'),
        headers: {
          'Authorization': 'Bearer demo_api_key',
        },
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        setState(() {
          sessionData = data;
          isLoading = false;
        });
      } else {
        setState(() {
          errorMessage = 'Failed to load payment session';
          isLoading = false;
        });
      }
    } catch (e) {
      setState(() {
        errorMessage = 'Error: \$e';
        isLoading = false;
      });
    }
  }

  Future<void> _simulatePayment(String status) async {
    setState(() => isLoading = true);

    try {
      final response = await http.post(
        Uri.parse('http://localhost:8081/pay/\${widget.sessionId}'),
        headers: {'Content-Type': 'application/json'},
        body: json.encode({'status': status}),
      );

      if (response.statusCode == 200) {
        _showResultDialog(status == 'succeeded' ? 'Payment Successful!' : 'Payment Failed!');
      } else {
        _showResultDialog('Error processing payment');
      }
    } catch (e) {
      _showResultDialog('Network error: \$e');
    }

    setState(() => isLoading = false);
  }

  void _showResultDialog(String message) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Payment Result'),
        content: Text(message),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.of(context).pop();
              Navigator.of(context).pop(); // Go back to main screen
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Payment Simulation'),
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
      ),
      body: isLoading 
        ? const Center(child: CircularProgressIndicator())
        : errorMessage != null
          ? Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(Icons.error, size: 64, color: Colors.red),
                  const SizedBox(height: 16),
                  Text(errorMessage!, style: const TextStyle(color: Colors.red)),
                ],
              ),
            )
          : Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Card(
                    child: Padding(
                      padding: const EdgeInsets.all(16.0),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text(
                            'Payment Details',
                            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                          ),
                          const SizedBox(height: 16),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              const Text('Amount:', style: TextStyle(fontSize: 16)),
                              Text(
                                '\${sessionData!['amount']} \${sessionData!['currency']}',
                                style: const TextStyle(
                                  fontSize: 24,
                                  fontWeight: FontWeight.bold,
                                  color: Colors.blue,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              const Text('Merchant:', style: TextStyle(fontSize: 16)),
                              Text(
                                sessionData!['business_name'] ?? 'Unknown',
                                style: const TextStyle(fontSize: 16),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  const Text(
                    'Simulate Payment Result:',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                  const SizedBox(height: 16),
                  ElevatedButton.icon(
                    onPressed: isLoading ? null : () => _simulatePayment('succeeded'),
                    icon: const Icon(Icons.check_circle, color: Colors.white),
                    label: const Text('Simulate Success', style: TextStyle(fontSize: 18)),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.green,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(vertical: 16),
                    ),
                  ),
                  const SizedBox(height: 12),
                  ElevatedButton.icon(
                    onPressed: isLoading ? null : () => _simulatePayment('failed'),
                    icon: const Icon(Icons.error, color: Colors.white),
                    label: const Text('Simulate Failure', style: TextStyle(fontSize: 18)),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.red,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(vertical: 16),
                    ),
                  ),
                ],
              ),
            ),
    );
  }
}