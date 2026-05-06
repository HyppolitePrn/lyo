import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../providers/auth_notifier.dart';
import '../widgets/auth_error_banner.dart';
import '../widgets/lyo_text_field.dart';

class RegisterScreen extends ConsumerStatefulWidget {
  const RegisterScreen({super.key});

  @override
  ConsumerState<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends ConsumerState<RegisterScreen> {
  final _formKey = GlobalKey<FormState>();
  final _firstNameCtrl = TextEditingController();
  final _lastNameCtrl = TextEditingController();
  final _usernameCtrl = TextEditingController();
  final _emailCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _confirmCtrl = TextEditingController();
  bool _agreed = false;

  static final _emailRegex = RegExp(r'^[^@]+@[^@]+\.[^@]+$');
  static final _usernameRegex = RegExp(r'^[a-zA-Z0-9_]{3,30}$');

  bool get _canSubmit =>
      _firstNameCtrl.text.trim().isNotEmpty &&
      _lastNameCtrl.text.trim().isNotEmpty &&
      _usernameCtrl.text.trim().isNotEmpty &&
      _emailCtrl.text.trim().isNotEmpty &&
      _passwordCtrl.text.length >= 8 &&
      _confirmCtrl.text == _passwordCtrl.text &&
      _agreed;

  @override
  void initState() {
    super.initState();
    for (final ctrl in [
      _firstNameCtrl,
      _lastNameCtrl,
      _usernameCtrl,
      _emailCtrl,
      _passwordCtrl,
      _confirmCtrl,
    ]) {
      ctrl.addListener(() => setState(() {}));
    }
  }

  @override
  void dispose() {
    _firstNameCtrl.dispose();
    _lastNameCtrl.dispose();
    _usernameCtrl.dispose();
    _emailCtrl.dispose();
    _passwordCtrl.dispose();
    _confirmCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    final success = await ref.read(authNotifierProvider.notifier).register(
          _usernameCtrl.text.trim(),
          _emailCtrl.text.trim(),
          _passwordCtrl.text,
        );
    if (mounted && success) {
      context.go('/home');
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authNotifierProvider);
    final sub = Theme.of(context).brightness == Brightness.dark
        ? lyoSubDark
        : lyoSubLight;

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
                  icon: const Icon(Icons.chevron_left),
                  padding: EdgeInsets.zero,
                  onPressed: () => context.pop(),
                ),
                const SizedBox(height: lyoGapXXL),
                _LogoMark(),
                const SizedBox(height: lyoGapXL),
                const Text(
                  'Create account',
                  style: TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.6,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  'Join Lyo and start listening',
                  style: TextStyle(fontSize: lyoBody1, color: sub),
                ),
                const SizedBox(height: lyoGapXXXL),
                Row(
                  children: [
                    Expanded(
                      child: LyoTextField(
                        controller: _firstNameCtrl,
                        hint: 'First name',
                        textInputAction: TextInputAction.next,
                        autofillHints: const [AutofillHints.givenName],
                        validator: (v) =>
                            (v == null || v.isEmpty) ? 'Required' : null,
                      ),
                    ),
                    const SizedBox(width: lyoGapM),
                    Expanded(
                      child: LyoTextField(
                        controller: _lastNameCtrl,
                        hint: 'Last name',
                        textInputAction: TextInputAction.next,
                        autofillHints: const [AutofillHints.familyName],
                        validator: (v) =>
                            (v == null || v.isEmpty) ? 'Required' : null,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 14),
                LyoTextField(
                  controller: _usernameCtrl,
                  hint: 'Username',
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.username],
                  validator: (v) {
                     if (v == null || v.isEmpty) {
                       return 'Username is required';
                     }
                     if (!_usernameRegex.hasMatch(v)) {
                       return '3–30 characters, letters, numbers or _';
                     }
                    return null;
                  },
                ),
                const SizedBox(height: 14),
                LyoTextField(
                  controller: _emailCtrl,
                  hint: 'Email address',
                  keyboardType: TextInputType.emailAddress,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.email],
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
                const SizedBox(height: 14),
                LyoTextField(
                  controller: _passwordCtrl,
                  hint: 'Password',
                  obscure: true,
                  textInputAction: TextInputAction.next,
                  autofillHints: const [AutofillHints.newPassword],
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
                  hint: 'Confirm password',
                  obscure: true,
                  textInputAction: TextInputAction.done,
                  autofillHints: const [AutofillHints.newPassword],
                  onFieldSubmitted: (_) => _canSubmit ? _submit() : null,
                  validator: (v) {
                    if (v == null || v.isEmpty) {
                      return 'Please confirm password';
                    }
                    if (v != _passwordCtrl.text) {
                      return 'Passwords do not match';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 10),
                _TermsRow(
                  agreed: _agreed,
                  onChanged: (v) => setState(() => _agreed = v ?? false),
                ),
                const SizedBox(height: 28),
                _CreateAccountButton(
                  isLoading: auth.isLoading,
                  enabled: _canSubmit,
                  onPressed: _submit,
                ),
                if (auth.error != null) ...[
                  const SizedBox(height: lyoGapM),
                  AuthErrorBanner(message: auth.error!),
                ],
                const SizedBox(height: lyoGapXXL),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(
                      'Already have an account? ',
                      style: TextStyle(fontSize: lyoBody2, color: sub),
                    ),
                    TextButton(
                      style: TextButton.styleFrom(
                        padding: EdgeInsets.zero,
                        minimumSize: Size.zero,
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      ),
                      onPressed: () => context.push('/login'),
                      child: const Text(
                        'Sign in',
                        style: TextStyle(
                          fontSize: lyoBody2,
                          fontWeight: FontWeight.w600,
                          color: lyoAccent,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: lyoGapXXL),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _CreateAccountButton extends StatelessWidget {
  const _CreateAccountButton({
    required this.isLoading,
    required this.enabled,
    required this.onPressed,
  });

  final bool isLoading;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return AnimatedOpacity(
      opacity: enabled ? 1.0 : 0.5,
      duration: const Duration(milliseconds: 200),
      child: DecoratedBox(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(lyoRadiusBtn),
          boxShadow: (enabled && !isLoading) ? const [lyoCtaGlow] : [],
        ),
        child: ElevatedButton(
          onPressed: (enabled && !isLoading) ? onPressed : null,
          child: isLoading
              ? const SizedBox(
                  width: 22,
                  height: 22,
                  child: CircularProgressIndicator(
                    color: Colors.white,
                    strokeWidth: 2,
                  ),
                )
              : const Text('Create Account'),
        ),
      ),
    );
  }
}

class _TermsRow extends StatelessWidget {
  const _TermsRow({required this.agreed, required this.onChanged});
  final bool agreed;
  final ValueChanged<bool?> onChanged;

  @override
  Widget build(BuildContext context) {
    final sub = Theme.of(context).brightness == Brightness.dark
        ? lyoSubDark
        : lyoSubLight;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Checkbox(
          value: agreed,
          onChanged: onChanged,
        ),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.only(top: 12),
            child: RichText(
              text: TextSpan(
                style: TextStyle(fontSize: lyoBody2, color: sub),
                children: [
                  const TextSpan(text: 'I agree to the '),
                  TextSpan(
                    text: 'Terms of Service',
                    style: const TextStyle(
                      color: lyoAccent,
                      fontWeight: FontWeight.w600,
                    ),
                    recognizer: TapGestureRecognizer()..onTap = () {},
                  ),
                  const TextSpan(text: ' and '),
                  TextSpan(
                    text: 'Privacy Policy',
                    style: const TextStyle(
                      color: lyoAccent,
                      fontWeight: FontWeight.w600,
                    ),
                    recognizer: TapGestureRecognizer()..onTap = () {},
                  ),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _LogoMark extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: lyoAccent,
        borderRadius: BorderRadius.circular(12),
      ),
      child: const Icon(Icons.radio, size: 20, color: Colors.white),
    );
  }
}
