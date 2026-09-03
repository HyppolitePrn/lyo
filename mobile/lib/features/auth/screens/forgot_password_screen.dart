import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../providers/auth_notifier.dart';
import '../widgets/auth_error_banner.dart';
import '../widgets/lyo_text_field.dart';

class ForgotPasswordScreen extends StatefulWidget {
  const ForgotPasswordScreen({super.key});

  @override
  State<ForgotPasswordScreen> createState() => _ForgotPasswordScreenState();
}

class _ForgotPasswordScreenState extends State<ForgotPasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailCtrl = TextEditingController();

  static final _emailRegex = RegExp(r'^[^@]+@[^@]+\.[^@]+$');

  @override
  void dispose() {
    _emailCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    await context.read<AuthNotifier>().forgotPassword(_emailCtrl.text.trim());
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthNotifier>();

    return Scaffold(
      resizeToAvoidBottomInset: true,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: lyoPadH),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SizedBox(height: lyoGapL),
                IconButton(
                  tooltip: 'Back',
                  icon: const Icon(Icons.chevron_left),
                  padding: EdgeInsets.zero,
                  onPressed: () => context.pop(),
                ),
                const SizedBox(height: lyoGapXXL),
                const Text(
                  'Forgot password?',
                  style: TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.6,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  "Enter your email and we'll send you a reset link",
                  style: TextStyle(
                    fontSize: lyoBody1,
                    color: Theme.of(context).brightness == Brightness.dark
                        ? lyoSubDark
                        : lyoSubLight,
                  ),
                ),
                const SizedBox(height: lyoGapXXXL),
                if (auth.resetEmailSent)
                  Text(
                    'Check your email for a link to reset your password.',
                    style: TextStyle(
                      fontSize: lyoBody1,
                      color: Theme.of(context).brightness == Brightness.dark
                          ? lyoTextDark
                          : lyoTextLight,
                    ),
                  )
                else ...[
                  LyoTextField(
                    controller: _emailCtrl,
                    hint: 'Email address',
                    keyboardType: TextInputType.emailAddress,
                    textInputAction: TextInputAction.done,
                    autofillHints: const [AutofillHints.email],
                    onFieldSubmitted: (_) => _submit(),
                    validator: (v) {
                      if (v == null || v.isEmpty) {
                        return 'Email is required';
                      }
                      if (!_emailRegex.hasMatch(v)) {
                        return 'Enter a valid email';
                      }
                      return null;
                    },
                  ),
                  const SizedBox(height: 28),
                  _SubmitButton(isLoading: auth.isLoading, onPressed: _submit),
                  if (auth.error != null) ...[
                    const SizedBox(height: lyoGapM),
                    AuthErrorBanner(message: auth.error!),
                  ],
                ],
                const SizedBox(height: lyoGapXXL),
              ],
            ),
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
            ? Semantics(
                label: 'Sending the reset link',
                liveRegion: true,
                child: const SizedBox(
                  width: 22,
                  height: 22,
                  child: CircularProgressIndicator(
                    color: Colors.white,
                    strokeWidth: 2,
                  ),
                ),
              )
            : const Text('Send reset link'),
      ),
    );
  }
}
