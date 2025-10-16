import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:wave_pool/api/api_client.dart';
import 'package:wave_pool/auth/auth_service.dart';
import 'package:wave_pool/auth/widgets/phone_input.dart';
import 'package:wave_pool/auth/widgets/pin_input.dart';
import 'package:wave_pool/core/secure_storage_service.dart';

class AuthScreen extends HookWidget {
  final AuthService? authService;
  const AuthScreen({super.key, this.authService});

  @override
  Widget build(BuildContext context) {
    final pageController = usePageController();
    final phone = useState('');
    final isLoading = useState(false);
    final authService = useMemoized(() {
      if (this.authService != null) {
        return this.authService!;
      }
      final secureStorage = SecureStorageService();
      final authSvc = AuthService(secureStorage, http.Client());
      return authSvc;
    }, [this.authService]);

    void handleAuth(String pin) async {
      isLoading.value = true;
      try {
        await authService.authenticate(phone.value, pin);
        if (context.mounted) {
          context.go('/dashboard');
        }
      } catch (e) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(e.toString())),
          );
        }
      } finally {
        isLoading.value = false;
      }
    }

    return Scaffold(
      body: Stack(
        children: [
          PageView(
            controller: pageController,
            physics: const NeverScrollableScrollPhysics(),
            children: [
              PhoneNumberInput(
                onNext: (phoneNumber) {
                  phone.value = phoneNumber;
                  pageController.nextPage(
                    duration: const Duration(milliseconds: 300),
                    curve: Curves.easeIn,
                  );
                },
              ),
              PinInputScreen(
                phone: phone.value,
                onCompleted: handleAuth,
                onBack: () {
                  pageController.previousPage(
                    duration: const Duration(milliseconds: 300),
                    curve: Curves.easeIn,
                  );
                },
              ),
            ],
          ),
          if (isLoading.value)
            const Center(
              child: CircularProgressIndicator(),
            ),
        ],
      ),
    );
  }
}