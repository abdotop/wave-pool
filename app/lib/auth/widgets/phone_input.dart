import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

class PhoneNumberInput extends HookWidget {
  final Function(String) onNext;
  const PhoneNumberInput({super.key, required this.onNext});

  @override
  Widget build(BuildContext context) {
    final phoneController = useTextEditingController();
    final phone = useState('');

    useEffect(() {
      void listener() => phone.value = phoneController.text;
      phoneController.addListener(listener);
      return () => phoneController.removeListener(listener);
    }, [phoneController]);

    final isValid = useMemoized(
      () => RegExp(r'^(77|78|75|71|70|76)[0-9]{7}$').hasMatch(phone.value),
      [phone.value],
    );

    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Column(
            children: [
              const SizedBox(height: 60),
              const Text(
                'Welcome to Wave Pool!',
                style: TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w600,
                  color: Colors.black,
                ),
              ),
              const SizedBox(height: 6),
              const Text(
                'Enter your mobile to start',
                style: TextStyle(fontSize: 15, color: Colors.black54),
              ),
              const SizedBox(height: 50),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    child: Row(
                      children: [
                        Image.asset(
                          'assets/flags/sn.jpg',
                          width: 28,
                          height: 20,
                          fit: BoxFit.cover,
                        ),
                        const SizedBox(width: 8),
                        const Text(
                          '+221',
                          style: TextStyle(fontSize: 18),
                        ),
                        const SizedBox(width: 6),
                        Container(
                          width: 1,
                          height: 22,
                          color: Colors.grey.shade400,
                        ),
                        const SizedBox(width: 6),
                      ],
                    ),
                  ),
                  Expanded(
                    child: TextField(
                      controller: phoneController,
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontSize: 20, letterSpacing: 1.5),
                      keyboardType: TextInputType.none,
                      decoration: const InputDecoration(
                        hintText: '7X XXX XX XX',
                        hintStyle: TextStyle(
                          color: Colors.grey,
                          fontSize: 20,
                          letterSpacing: 1.5,
                        ),
                        border: UnderlineInputBorder(
                          borderSide: BorderSide(color: Colors.blueAccent),
                        ),
                        focusedBorder: UnderlineInputBorder(
                          borderSide: BorderSide(color: Colors.lightBlueAccent),
                        ),
                      ),
                      maxLength: 9,
                    ),
                  ),
                ],
              ),

              const Spacer(),

              Numpad(
                onDigit: (d) {
                  if (phoneController.text.length < 9) {
                    phoneController.text += d;
                  }
                },
                onBackspace: () {
                  final text = phoneController.text;
                  if (text.isNotEmpty) {
                    phoneController.text = text.substring(0, text.length - 1);
                  }
                },
              ),
              const SizedBox(height: 16),

              SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    elevation: 0,
                    backgroundColor:
                        isValid ? const Color(0xFFB3E5FC) : Colors.grey[200],
                    foregroundColor: Colors.black,
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(50),
                    ),
                  ),
                  onPressed: isValid ? () => onNext(phone.value) : null,
                  child: const Text(
                    'Next',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.w500),
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

class Numpad extends StatelessWidget {
  const Numpad({super.key, required this.onDigit, required this.onBackspace});
  final ValueSetter<String> onDigit;
  final VoidCallback onBackspace;

  @override
  Widget build(BuildContext context) {
    return GridView.count(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      crossAxisCount: 3,
      childAspectRatio: 1.3,
      mainAxisSpacing: 12,
      crossAxisSpacing: 12,
      children: [
        for (final d in ['1', '2', '3', '4', '5', '6', '7', '8', '9'])
          _NumpadButton(label: d, onTap: () => onDigit(d)),
        const SizedBox.shrink(),
        _NumpadButton(label: '0', onTap: () => onDigit('0')),
        _NumpadButton(label: '⌫', onTap: onBackspace),
      ],
    );
  }
}

class _NumpadButton extends StatelessWidget {
  const _NumpadButton({required this.label, required this.onTap});
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Center(
        child: Text(
          label,
          style: const TextStyle(fontSize: 26, fontWeight: FontWeight.w400),
        ),
      ),
    );
  }
}
