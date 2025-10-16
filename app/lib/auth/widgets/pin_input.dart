import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:wave_pool/auth/widgets/phone_input.dart';

class PinInputScreen extends HookWidget {
  final String phone;
  final Function(String) onCompleted;
  final VoidCallback onBack;

  const PinInputScreen({
    super.key,
    required this.phone,
    required this.onCompleted,
    required this.onBack,
  });

  @override
  Widget build(BuildContext context) {
    final pinController = useTextEditingController();

    useEffect(() {
      void listener() {
        if (pinController.text.length == 4) {
          onCompleted(pinController.text);
        }
      }

      pinController.addListener(listener);
      return () => pinController.removeListener(listener);
    }, [pinController]);

    void addDigit(String d) {
      if (pinController.text.length < 4) {
        pinController.text += d;
      }
    }

    void removeDigit() {
      final text = pinController.text;
      if (text.isNotEmpty) {
        pinController.text = text.substring(0, text.length - 1);
      }
    }

    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Column(
            children: [
              const SizedBox(height: 16),
              Align(
                alignment: Alignment.centerLeft,
                child: IconButton(
                  icon: const Icon(Icons.arrow_back, color: Colors.black),
                  onPressed: onBack,
                ),
              ),
              const SizedBox(height: 30),
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: Colors.blue.shade50,
                ),
                child: const Icon(Icons.lock_outline,
                    color: Colors.blueAccent, size: 42),
              ),
              const SizedBox(height: 20),
              const Text(
                'Enter your secret code for account',
                style: TextStyle(
                  fontSize: 16,
                  color: Colors.black87,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                phone.replaceAllMapped(
                    RegExp(r"(\d{2})(\d{3})(\d{2})(\d{2})"),
                    (m) => "${m[1]} ${m[2]} ${m[3]} ${m[4]}"),
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w500,
                  letterSpacing: 1.2,
                ),
              ),
              const SizedBox(height: 40),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: List.generate(4, (i) {
                  final filled = i < pinController.text.length;
                  return AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    margin: const EdgeInsets.symmetric(horizontal: 10),
                    width: 16,
                    height: 16,
                    decoration: BoxDecoration(
                      color: filled ? Colors.lightBlueAccent : Colors.grey[300],
                      shape: BoxShape.circle,
                    ),
                  );
                }),
              ),
              const Spacer(),
              Numpad(
                onDigit: addDigit,
                onBackspace: removeDigit,
              ),
              const SizedBox(height: 12),
              Align(
                alignment: Alignment.centerLeft,
                child: GestureDetector(
                  onTap: () {
                    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                        content: Text('Forgot PIN tapped')));
                  },
                  child: const Padding(
                    padding: EdgeInsets.only(left: 4, top: 8),
                    child: Text(
                      'FORGOT?',
                      style: TextStyle(
                        fontWeight: FontWeight.w500,
                        color: Colors.black54,
                        letterSpacing: 0.8,
                      ),
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }
}
