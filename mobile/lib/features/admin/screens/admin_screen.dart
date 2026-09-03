import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../providers/admin_feature_notifier.dart';

class AdminScreen extends StatefulWidget {
  const AdminScreen({super.key});

  @override
  State<AdminScreen> createState() => _AdminScreenState();
}

class _AdminScreenState extends State<AdminScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final token = context.read<AuthNotifier>().accessToken;
      if (token != null) {
        context.read<AdminFeatureNotifier>().load(token);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    final role = context.watch<AuthNotifier>().role;
    if (role != 'admin') {
      return Scaffold(
        backgroundColor: bg,
        body: Center(
          child: Text('Not authorized', style: TextStyle(color: textSub)),
        ),
      );
    }

    final notifier = context.watch<AdminFeatureNotifier>();

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        title: Text('Feature Flags',
            style: TextStyle(
                color: textPrimary, fontSize: lyoH1, fontWeight: FontWeight.w700)),
      ),
      body: switch (notifier.status) {
        AdminFeatureStatus.loading || AdminFeatureStatus.idle =>
          const Center(child: CircularProgressIndicator(color: lyoAccent)),
        AdminFeatureStatus.error => Center(
            child: Text(notifier.error ?? 'Something went wrong',
                style: const TextStyle(color: lyoError))),
        AdminFeatureStatus.ready => ListView.separated(
            padding: const EdgeInsets.all(lyoGapM),
            itemCount: notifier.flags.length,
            separatorBuilder: (_, _) => const SizedBox(height: lyoGapS),
            itemBuilder: (context, i) {
              final flag = notifier.flags[i];
              return SwitchListTile(
                activeThumbColor: lyoAccent,
                title: Text(flag.name,
                    style: TextStyle(color: textPrimary, fontWeight: FontWeight.w600)),
                subtitle: Text(flag.description, style: TextStyle(color: textSub)),
                value: flag.enabled,
                onChanged: (value) {
                  final token = context.read<AuthNotifier>().accessToken;
                  if (token == null) {
                    return;
                  }
                  context
                      .read<AdminFeatureNotifier>()
                      .toggle(name: flag.name, enabled: value, token: token);
                },
              );
            },
          ),
      },
    );
  }
}
