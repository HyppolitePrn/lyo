import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../providers/auth_notifier.dart';
import '../widgets/auth_error_banner.dart';
import '../widgets/lyo_text_field.dart';

class ResetPasswordScreen extends StatefulWidget {
  const ResetPasswordScreen({required this.token, super.key});

  final String? token;

  @override
  State<ResetPasswordScreen> createState() => _ResetPasswordScreenState();
}

class _ResetPasswordScreenState extends State<ResetPasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  final _passwordCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();

  @override
  void dispose() {
    _passwordCtrl.dispose();
    _confirmCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    final token = widget.token;
    if (token == null) {
      return;
    }
    final success = await context
        .read<AuthNotifier>()
        .resetPassword(token, _passwordCtrl.text);
    if (mounted && success) {
      context.go('/login');
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthNotifier>();

    return Scaffold(
      resizeToAvoidBottomInset: true,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: lyoPadH),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SizedBox(height: lyoGapL),
              IconButton(
                icon: const Icon(Icons.chevron_left),
                padding: EdgeInsets.zero,
                onPressed: () => context.pop(),
              ),
              const SizedBox(height: lyoGapXXL),
              const Text(
                'Reset password',
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.w800,
                  letterSpacing: -0.6,
                ),
              ),
              const SizedBox(height: lyoGapXXXL),
              if (widget.token == null)
                const AuthErrorBanner(
                  message:
                      'This reset link is invalid or missing a token. Request a new one.',
                )
              else
                Form(
                  key: _formKey,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      LyoTextField(
                        controller: _passwordCtrl,
                        hint: 'New password',
                        obscure: true,
                        textInputAction: TextInputAction.next,
                        validator: (v) {
                          if (v == null || v.isEmpty) {
                            return 'Password is required';
                          }
                          if (v.length < 8) {
                            return 'At least 8 characters';
                          }
                          return null;
                        },
                      ),
                      const SizedBox(height: 14),
                      LyoTextField(
                        controller: _confirmCtrl,
                        hint: 'Confirm new password',
                        obscure: true,
                        textInputAction: TextInputAction.done,
                        onFieldSubmitted: (_) => _submit(),
                        validator: (v) {
                          if (v != _passwordCtrl.text) {
                            return 'Passwords do not match';
                          }
                          return null;
                        },
                      ),
                      const SizedBox(height: 28),
                      _SubmitButton(
                          isLoading: auth.isLoading, onPressed: _submit),
                      if (auth.error != null) ...[
                        const SizedBox(height: lyoGapM),
                        AuthErrorBanner(message: auth.error!),
                      ],
                    ],
                  ),
                ),
              const SizedBox(height: lyoGapXXL),
            ],
          ),
        ),
      ),
    );
  }
}

class _SubmitButton extends StatelessWidget {
  const _SubmitButton({required this.isLoading, required this.onPressed});
  final bool isLoading;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(lyoRadiusBtn),
        boxShadow: isLoading ? [] : const [lyoCtaGlow],
      ),
      child: ElevatedButton(
        onPressed: isLoading ? null : onPressed,
        child: isLoading
            ? const SizedBox(
                width: 22,
                height: 22,
                child: CircularProgressIndicator(
                  color: Colors.white,
                  strokeWidth: 2,
                ),
              )
            : const Text('Reset password'),
      ),
    );
  }
}
