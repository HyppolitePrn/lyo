import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class LyoTextField extends StatefulWidget {
  const LyoTextField({
    required this.controller, required this.hint, super.key,
    this.label,
    this.obscure = false,
    this.keyboardType,
    this.validator,
    this.textInputAction,
    this.onFieldSubmitted,
    this.prefixIcon,
    this.autofillHints,
  });

  final TextEditingController controller;
  final String hint;
  final String? label;
  final bool obscure;
  final TextInputType? keyboardType;
  final String? Function(String?)? validator;
  final TextInputAction? textInputAction;
  final void Function(String)? onFieldSubmitted;
  final Widget? prefixIcon;
  final Iterable<String>? autofillHints;

  @override
  State<LyoTextField> createState() => _LyoTextFieldState();
}

class _LyoTextFieldState extends State<LyoTextField> {
  late bool _obscured;

  @override
  void initState() {
    super.initState();
    _obscured = widget.obscure;
  }

  @override
  Widget build(BuildContext context) {
    return TextFormField(
      controller: widget.controller,
      obscureText: _obscured,
      keyboardType: widget.keyboardType,
      textInputAction: widget.textInputAction,
      onFieldSubmitted: widget.onFieldSubmitted,
      autofillHints: widget.autofillHints,
      style: const TextStyle(fontSize: lyoBody1),
      validator: widget.validator,
      decoration: InputDecoration(
        hintText: widget.hint,
        labelText: widget.label,
        prefixIcon: widget.prefixIcon,
        suffixIcon: widget.obscure
            ? IconButton(
                icon: Icon(
                  _obscured ? Icons.visibility_off_outlined : Icons.visibility_outlined,
                  size: 20,
                  color: lyoSubDark,
                ),
                onPressed: () => setState(() => _obscured = !_obscured),
              )
            : null,
      ),
    );
  }
}
