import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../auth/widgets/auth_error_banner.dart';
import '../../auth/widgets/lyo_text_field.dart';
import '../providers/upload_track_notifier.dart';

class UploadTrackScreen extends StatefulWidget {
  const UploadTrackScreen({super.key});

  @override
  State<UploadTrackScreen> createState() => _UploadTrackScreenState();
}

class _UploadTrackScreenState extends State<UploadTrackScreen> {
  final _titleCtrl = TextEditingController();
  final _artistCtrl = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _titleCtrl.dispose();
    _artistCtrl.dispose();
    super.dispose();
  }

  Future<void> _pickFile() async {
    final result = await FilePicker.platform.pickFiles(type: FileType.audio);
    final path = result?.files.single.path;
    if (path != null && mounted) {
      context.read<UploadTrackNotifier>().selectFile(File(path));
    }
  }

  Future<void> _upload() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    final token = context.read<AuthNotifier>().accessToken ?? '';
    await context.read<UploadTrackNotifier>().upload(
          title: _titleCtrl.text.trim(),
          artist: _artistCtrl.text.trim().isEmpty ? null : _artistCtrl.text.trim(),
          token: token,
        );
  }

  @override
  Widget build(BuildContext context) {
    final upload = context.watch<UploadTrackNotifier>();
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final isBusy = upload.status == UploadTrackStatus.uploading;

    if (upload.status == UploadTrackStatus.success) {
      return Scaffold(
        backgroundColor: bg,
        appBar: AppBar(backgroundColor: bg, elevation: 0),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.check_circle, size: 56, color: lyoAccent),
              const SizedBox(height: lyoGapM),
              Text('Track uploaded',
                  style: TextStyle(
                      color: textPrimary,
                      fontSize: lyoH1,
                      fontWeight: FontWeight.w700)),
              const SizedBox(height: lyoGapS),
              Text(upload.uploadedTrack?.title ?? '',
                  style: TextStyle(color: textSub, fontSize: lyoBody2)),
              const SizedBox(height: lyoGapXL),
              ElevatedButton(
                onPressed: () {
                  upload.reset();
                  Navigator.of(context).pop();
                },
                style: ElevatedButton.styleFrom(
                  backgroundColor: lyoAccent,
                  foregroundColor: Colors.white,
                ),
                child: const Text('Done'),
              ),
            ],
          ),
        ),
      );
    }

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        iconTheme: IconThemeData(color: textPrimary),
        title: Text(
          'Upload Track',
          style: TextStyle(
              color: textPrimary, fontSize: lyoH1, fontWeight: FontWeight.w700),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(
              horizontal: lyoPadHMain, vertical: lyoGapXL),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (upload.error != null) AuthErrorBanner(message: upload.error!),
              GestureDetector(
                onTap: isBusy ? null : _pickFile,
                child: Container(
                  padding: const EdgeInsets.all(lyoGapL),
                  decoration: BoxDecoration(
                    color: surface,
                    borderRadius: BorderRadius.circular(lyoRadiusCard),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.audio_file_outlined,
                          size: 28, color: lyoAccent),
                      const SizedBox(width: lyoGapM),
                      Expanded(
                        child: Text(
                          upload.selectedFile?.path.split('/').last ??
                              'Choose an audio file',
                          style: TextStyle(color: textPrimary, fontSize: lyoBody2),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: lyoGapXXL),
              Form(
                key: _formKey,
                child: Column(
                  children: [
                    LyoTextField(
                      controller: _titleCtrl,
                      hint: 'Track title',
                      label: 'Title',
                      textInputAction: TextInputAction.next,
                      validator: (v) => (v == null || v.trim().isEmpty)
                          ? 'Title is required'
                          : null,
                    ),
                    const SizedBox(height: lyoGapL),
                    LyoTextField(
                      controller: _artistCtrl,
                      hint: 'Artist (optional)',
                      label: 'Artist',
                      textInputAction: TextInputAction.done,
                    ),
                  ],
                ),
              ),
              const SizedBox(height: lyoGapXXXL),
              isBusy
                  ? const Center(
                      child: CircularProgressIndicator(color: lyoAccent))
                  : ElevatedButton(
                      onPressed: _upload,
                      style: ElevatedButton.styleFrom(
                        backgroundColor: lyoAccent,
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(vertical: lyoGapM),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(lyoRadiusBtn),
                        ),
                        elevation: 0,
                      ),
                      child: const Text(
                        'Upload',
                        style: TextStyle(
                            fontSize: lyoBody1,
                            fontWeight: FontWeight.w700,
                            letterSpacing: 0.5),
                      ),
                    ),
            ],
          ),
        ),
      ),
    );
  }
}
