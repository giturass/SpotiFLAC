import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:file_picker/file_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:go_router/go_router.dart';
import 'package:device_info_plus/device_info_plus.dart';
import 'package:spotiflac_android/providers/settings_provider.dart';

class SetupScreen extends ConsumerStatefulWidget {
  const SetupScreen({super.key});

  @override
  ConsumerState<SetupScreen> createState() => _SetupScreenState();
}

class _SetupScreenState extends ConsumerState<SetupScreen> {
  int _currentStep = 0;
  bool _permissionGranted = false;
  String? _selectedDirectory;
  bool _isLoading = false;
  int _androidSdkVersion = 0;

  @override
  void initState() {
    super.initState();
    _initDeviceInfo();
  }

  Future<void> _initDeviceInfo() async {
    if (Platform.isAndroid) {
      final deviceInfo = DeviceInfoPlugin();
      final androidInfo = await deviceInfo.androidInfo;
      _androidSdkVersion = androidInfo.version.sdkInt;
      debugPrint('Android SDK Version: $_androidSdkVersion');
    }
    await _checkInitialPermission();
  }

  Future<void> _checkInitialPermission() async {
    if (Platform.isIOS) {
      // iOS doesn't need storage permission - app uses its own Documents directory
      if (mounted) {
        setState(() => _permissionGranted = true);
      }
    } else if (Platform.isAndroid) {
      PermissionStatus status;
      
      if (_androidSdkVersion >= 33) {
        status = await Permission.audio.status;
      } else if (_androidSdkVersion >= 30) {
        status = await Permission.manageExternalStorage.status;
      } else {
        status = await Permission.storage.status;
      }
      
      if (status.isGranted && mounted) {
        setState(() => _permissionGranted = true);
      }
    }
  }

  Future<void> _requestPermission() async {
    setState(() => _isLoading = true);

    try {
      if (Platform.isIOS) {
        // iOS doesn't need storage permission - app uses its own Documents directory
        setState(() => _permissionGranted = true);
      } else if (Platform.isAndroid) {
        PermissionStatus status;
        
        if (_androidSdkVersion >= 33) {
          status = await Permission.audio.request();
          if (!status.isGranted) {
            await Permission.notification.request();
          }
        } else if (_androidSdkVersion >= 30) {
          status = await Permission.manageExternalStorage.request();
        } else {
          status = await Permission.storage.request();
        }
        
        if (status.isGranted) {
          setState(() => _permissionGranted = true);
        } else if (status.isPermanentlyDenied) {
          _showPermissionDeniedDialog();
        } else {
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('Permission denied. Please grant permission to continue.'),
              ),
            );
          }
        }
      }
    } catch (e) {
      debugPrint('Permission error: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  void _showPermissionDeniedDialog() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Permission Required'),
        content: const Text(
          'Storage permission is required to save downloaded music files. '
          'Please grant permission in app settings.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(context);
              openAppSettings();
            },
            child: const Text('Open Settings'),
          ),
        ],
      ),
    );
  }

  Future<void> _selectDirectory() async {
    setState(() => _isLoading = true);

    try {
      String? selectedDirectory = await FilePicker.platform.getDirectoryPath(
        dialogTitle: 'Select Download Folder',
      );

      if (selectedDirectory != null) {
        setState(() => _selectedDirectory = selectedDirectory);
      } else {
        final defaultDir = await _getDefaultDirectory();
        if (mounted) {
          final useDefault = await showDialog<bool>(
            context: context,
            builder: (context) => AlertDialog(
              title: const Text('Use Default Folder?'),
              content: Text(
                'No folder selected. Would you like to use the default Music folder?\n\n$defaultDir',
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.pop(context, false),
                  child: const Text('Cancel'),
                ),
                TextButton(
                  onPressed: () => Navigator.pop(context, true),
                  child: const Text('Use Default'),
                ),
              ],
            ),
          );

          if (useDefault == true) {
            setState(() => _selectedDirectory = defaultDir);
          }
        }
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  Future<String> _getDefaultDirectory() async {
    if (Platform.isIOS) {
      // iOS: Use Documents directory (accessible via Files app)
      final appDir = await getApplicationDocumentsDirectory();
      final musicDir = Directory('${appDir.path}/SpotiFLAC');
      try {
        if (!await musicDir.exists()) {
          await musicDir.create(recursive: true);
        }
        return musicDir.path;
      } catch (e) {
        debugPrint('Cannot create SpotiFLAC folder: $e');
      }
      return '${appDir.path}/SpotiFLAC';
    } else if (Platform.isAndroid) {
      final musicDir = Directory('/storage/emulated/0/Music/SpotiFLAC');
      try {
        if (!await musicDir.exists()) {
          await musicDir.create(recursive: true);
        }
        return musicDir.path;
      } catch (e) {
        debugPrint('Cannot create Music folder: $e');
      }
    }
    final appDir = await getApplicationDocumentsDirectory();
    return '${appDir.path}/SpotiFLAC';
  }

  Future<void> _completeSetup() async {
    if (_selectedDirectory == null) return;

    setState(() => _isLoading = true);

    try {
      final dir = Directory(_selectedDirectory!);
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }

      ref.read(settingsProvider.notifier).setDownloadDirectory(_selectedDirectory!);
      ref.read(settingsProvider.notifier).setFirstLaunchComplete();

      if (mounted) {
        context.go('/');
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    
    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24.0),
          child: ConstrainedBox(
            constraints: BoxConstraints(
              minHeight: MediaQuery.of(context).size.height -
                  MediaQuery.of(context).padding.top -
                  MediaQuery.of(context).padding.bottom - 48,
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                // Top section - Logo/Title
                Column(
                  children: [
                    const SizedBox(height: 24),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(24),
                      child: Image.asset(
                        'assets/images/logo.png',
                        width: 96,
                        height: 96,
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'SpotiFLAC',
                      style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                        color: colorScheme.primary,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'Download Spotify tracks in FLAC',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),

                // Middle section - Steps and Content
                Column(
                  children: [
                    const SizedBox(height: 24),
                    _buildStepIndicator(colorScheme),
                    const SizedBox(height: 24),
                    _currentStep == 0
                        ? _buildPermissionStep(colorScheme)
                        : _buildDirectoryStep(colorScheme),
                  ],
                ),

                // Bottom section - Navigation Buttons
                Column(
                  children: [
                    const SizedBox(height: 24),
                    _buildNavigationButtons(colorScheme),
                    const SizedBox(height: 16),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildStepIndicator(ColorScheme colorScheme) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        _buildStepDot(0, 'Permission', colorScheme),
        Container(
          width: 40,
          height: 2,
          color: _currentStep >= 1 ? colorScheme.primary : colorScheme.surfaceContainerHighest,
        ),
        _buildStepDot(1, 'Folder', colorScheme),
      ],
    );
  }

  Widget _buildStepDot(int step, String label, ColorScheme colorScheme) {
    final isActive = _currentStep >= step;
    final isCompleted = (step == 0 && _permissionGranted) ||
        (step == 1 && _selectedDirectory != null);

    return Column(
      children: [
        Container(
          width: 32,
          height: 32,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: isCompleted
                ? colorScheme.primary
                : isActive
                    ? colorScheme.primaryContainer
                    : colorScheme.surfaceContainerHighest,
          ),
          child: Center(
            child: isCompleted
                ? Icon(Icons.check, size: 18, color: colorScheme.onPrimary)
                : Text(
                    '${step + 1}',
                    style: TextStyle(
                      color: isActive ? colorScheme.onPrimaryContainer : colorScheme.onSurfaceVariant,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          label,
          style: Theme.of(context).textTheme.labelSmall?.copyWith(
            color: isActive ? colorScheme.onSurface : colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }

  Widget _buildPermissionStep(ColorScheme colorScheme) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          _permissionGranted ? Icons.check_circle : Icons.folder_open,
          size: 56,
          color: _permissionGranted ? colorScheme.primary : colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 16),
        Text(
          _permissionGranted
              ? 'Storage Permission Granted!'
              : 'Storage Permission Required',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          _permissionGranted
              ? 'You can now select where to save your music files.'
              : 'SpotiFLAC needs storage access to save downloaded music files to your device.',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 20),
        if (!_permissionGranted)
          FilledButton.icon(
            onPressed: _isLoading ? null : _requestPermission,
            icon: _isLoading
                ? SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: colorScheme.onPrimary,
                    ),
                  )
                : const Icon(Icons.security),
            label: const Text('Grant Permission'),
            style: FilledButton.styleFrom(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
            ),
          ),
      ],
    );
  }

  Widget _buildDirectoryStep(ColorScheme colorScheme) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          _selectedDirectory != null ? Icons.folder : Icons.create_new_folder,
          size: 56,
          color: _selectedDirectory != null ? colorScheme.primary : colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 16),
        Text(
          _selectedDirectory != null
              ? 'Download Folder Selected!'
              : 'Choose Download Folder',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        if (_selectedDirectory != null)
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.folder, color: colorScheme.primary, size: 20),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    _selectedDirectory!,
                    style: Theme.of(context).textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          )
        else
          Text(
            'Select a folder where your downloaded music will be saved.',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
        const SizedBox(height: 20),
        FilledButton.icon(
          onPressed: _isLoading ? null : _selectDirectory,
          icon: _isLoading
              ? SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: colorScheme.onPrimary,
                  ),
                )
              : Icon(_selectedDirectory != null ? Icons.edit : Icons.folder_open),
          label: Text(_selectedDirectory != null ? 'Change Folder' : 'Select Folder'),
          style: FilledButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
          ),
        ),
      ],
    );
  }

  Widget _buildNavigationButtons(ColorScheme colorScheme) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        // Back button
        if (_currentStep > 0)
          TextButton.icon(
            onPressed: () => setState(() => _currentStep--),
            icon: const Icon(Icons.arrow_back),
            label: const Text('Back'),
          )
        else
          const SizedBox(width: 100),

        // Next/Finish button
        if (_currentStep == 0)
          FilledButton(
            onPressed: _permissionGranted
                ? () => setState(() => _currentStep++)
                : null,
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Next'),
                SizedBox(width: 8),
                Icon(Icons.arrow_forward, size: 18),
              ],
            ),
          )
        else
          FilledButton(
            onPressed: _selectedDirectory != null && !_isLoading
                ? _completeSetup
                : null,
            child: _isLoading
                ? SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(
                      strokeWidth: 2, 
                      color: colorScheme.onPrimary,
                    ),
                  )
                : const Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text('Get Started'),
                      SizedBox(width: 8),
                      Icon(Icons.check, size: 18),
                    ],
                  ),
          ),
      ],
    );
  }
}
