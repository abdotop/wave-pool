import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:wave_pool/auth/auth_screen.dart';
import 'package:wave_pool/core/secure_storage_service.dart';
import 'package:wave_pool/dashboard/dashboard_screen.dart';
import 'package:wave_pool/splash/splash_screen.dart';

class AppRouter {
  final SecureStorageService secureStorageService;

  AppRouter({required this.secureStorageService});

  late final GoRouter router = GoRouter(
    initialLocation: '/splash',
    routes: <GoRoute>[
      GoRoute(
        path: '/splash',
        builder: (BuildContext context, GoRouterState state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/auth',
        builder: (BuildContext context, GoRouterState state) => const AuthScreen(),
      ),
      GoRoute(
        path: '/dashboard',
        builder: (BuildContext context, GoRouterState state) => const DashboardScreen(),
      ),
    ],
    redirect: (BuildContext context, GoRouterState state) async {
      final isAuthenticated = await secureStorageService.hasToken();
      final isAuthRoute = state.matchedLocation == '/auth';
      final isSplashRoute = state.matchedLocation == '/splash';
      final isCheckingAuth = state.matchedLocation == '/check-auth';

      if (isSplashRoute) {
        return null;
      }

      if (!isAuthenticated && !isAuthRoute) {
        return '/auth';
      }

      if (isAuthenticated && (isAuthRoute || isCheckingAuth)) {
        return '/dashboard';
      }

      return null;
    },
  );
}