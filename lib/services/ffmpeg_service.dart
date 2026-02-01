import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:ffmpeg_kit_flutter_new_audio/ffmpeg_kit.dart';
import 'package:ffmpeg_kit_flutter_new_audio/ffmpeg_kit_config.dart';
import 'package:ffmpeg_kit_flutter_new_audio/return_code.dart';
import 'package:path_provider/path_provider.dart';
import 'package:spotiflac_android/utils/logger.dart';

final _log = AppLogger('FFmpeg');

/// FFmpeg service for audio conversion and remuxing
/// Uses ffmpeg_kit_flutter_new_audio plugin
class FFmpegService {
  static Future<FFmpegResult> _execute(String command) async {
    try {
      final session = await FFmpegKit.execute(command);
      final returnCode = await session.getReturnCode();
      final output = await session.getOutput() ?? '';
      
      return FFmpegResult(
        success: ReturnCode.isSuccess(returnCode),
        returnCode: returnCode?.getValue() ?? -1,
        output: output,
      );
    } catch (e) {
      _log.e('FFmpeg execute error: $e');
      return FFmpegResult(success: false, returnCode: -1, output: e.toString());
    }
  }

  static Future<String?> convertM4aToFlac(String inputPath) async {
    final outputPath = inputPath.replaceAll('.m4a', '.flac');

    final command =
        '-i "$inputPath" -c:a flac -compression_level 8 "$outputPath" -y';

    final result = await _execute(command);

    if (result.success) {
      try {
        await File(inputPath).delete();
      } catch (_) {}
      return outputPath;
    }

    _log.e('M4A to FLAC conversion failed: ${result.output}');
    return null;
  }

  /// Convert M4A (AAC) to lossy format (MP3 or Opus)
  /// format: 'mp3' or 'opus'
  /// bitrate: e.g., 'mp3_320', 'opus_128' - extracts the kbps value
  static Future<String?> convertM4aToLossy(
    String inputPath, {
    required String format,
    String? bitrate,
    bool deleteOriginal = true,
  }) async {
    // Extract bitrate value from format like 'mp3_320' -> '320k'
    String bitrateValue = format == 'opus' ? '128k' : '320k';
    if (bitrate != null && bitrate.contains('_')) {
      final parts = bitrate.split('_');
      if (parts.length == 2) {
        bitrateValue = '${parts[1]}k';
      }
    }
    
    final extension = format == 'opus' ? '.opus' : '.mp3';
    final outputPath = inputPath.replaceAll('.m4a', extension);
    
    String command;
    if (format == 'opus') {
      // M4A -> Opus conversion
      command =
          '-i "$inputPath" -codec:a libopus -b:a $bitrateValue -vbr on -compression_level 10 -map 0:a "$outputPath" -y';
    } else {
      // M4A -> MP3 conversion
      command =
          '-i "$inputPath" -codec:a libmp3lame -b:a $bitrateValue -map 0:a -id3v2_version 3 "$outputPath" -y';
    }

    final result = await _execute(command);

    if (result.success) {
      if (deleteOriginal) {
        try {
          await File(inputPath).delete();
        } catch (_) {}
      }
      return outputPath;
    }

    _log.e('M4A to $format conversion failed: ${result.output}');
    return null;
  }

  static Future<String?> convertFlacToMp3(
    String inputPath, {
    String bitrate = '320k',
    bool deleteOriginal = true,
  }) async {
    final outputPath = inputPath.replaceAll('.flac', '.mp3');

    final command =
        '-i "$inputPath" -codec:a libmp3lame -b:a $bitrate -map 0:a -map_metadata 0 -id3v2_version 3 "$outputPath" -y';

    final result = await _execute(command);

    if (result.success) {
      if (deleteOriginal) {
        try {
          await File(inputPath).delete();
        } catch (_) {}
      }
      return outputPath;
    }

    _log.e('FLAC to MP3 conversion failed: ${result.output}');
    return null;
  }

  static Future<String?> convertFlacToOpus(
    String inputPath, {
    String bitrate = '128k',
    bool deleteOriginal = true,
  }) async {
    final outputPath = inputPath.replaceAll('.flac', '.opus');

    // Opus in OGG container with VBR
    final command =
        '-i "$inputPath" -codec:a libopus -b:a $bitrate -vbr on -compression_level 10 -map 0:a -map_metadata 0 "$outputPath" -y';

    final result = await _execute(command);

    if (result.success) {
      if (deleteOriginal) {
        try {
          await File(inputPath).delete();
        } catch (_) {}
      }
      return outputPath;
    }

    _log.e('FLAC to Opus conversion failed: ${result.output}');
    return null;
  }

  /// Convert FLAC to lossy format based on format parameter
  /// format: 'mp3' or 'opus'
  /// bitrate: e.g., 'mp3_320', 'opus_128' - extracts the kbps value
  static Future<String?> convertFlacToLossy(
    String inputPath, {
    required String format,
    String? bitrate,
    bool deleteOriginal = true,
  }) async {
    // Extract bitrate value from format like 'mp3_320' -> '320k'
    String bitrateValue = '320k'; // default for mp3
    if (bitrate != null && bitrate.contains('_')) {
      final parts = bitrate.split('_');
      if (parts.length == 2) {
        bitrateValue = '${parts[1]}k';
      }
    }
    
    switch (format.toLowerCase()) {
      case 'opus':
        final opusBitrate = bitrate?.startsWith('opus_') == true ? bitrateValue : '128k';
        return convertFlacToOpus(inputPath, bitrate: opusBitrate, deleteOriginal: deleteOriginal);
      case 'mp3':
      default:
        final mp3Bitrate = bitrate?.startsWith('mp3_') == true ? bitrateValue : '320k';
        return convertFlacToMp3(inputPath, bitrate: mp3Bitrate, deleteOriginal: deleteOriginal);
    }
  }

  static Future<String?> convertFlacToM4a(
    String inputPath, {
    String codec = 'aac',
    String bitrate = '256k',
  }) async {
    final dir = File(inputPath).parent.path;
    final baseName =
        inputPath.split(Platform.pathSeparator).last.replaceAll('.flac', '');
    final outputDir = '$dir${Platform.pathSeparator}M4A';

    await Directory(outputDir).create(recursive: true);

    final outputPath = '$outputDir${Platform.pathSeparator}$baseName.m4a';

    String command;
    if (codec == 'alac') {
      command =
          '-i "$inputPath" -codec:a alac -map 0:a -map_metadata 0 "$outputPath" -y';
    } else {
      command =
          '-i "$inputPath" -codec:a aac -b:a $bitrate -map 0:a -map_metadata 0 "$outputPath" -y';
    }

    final result = await _execute(command);

    if (result.success) {
      return outputPath;
    }

    _log.e('FLAC to M4A conversion failed: ${result.output}');
    return null;
  }

  static Future<bool> isAvailable() async {
    try {
      final version = await FFmpegKitConfig.getFFmpegVersion();
      return version?.isNotEmpty ?? false;
    } catch (e) {
      return false;
    }
  }

  static Future<String?> getVersion() async {
    try {
      return await FFmpegKitConfig.getFFmpegVersion();
    } catch (e) {
      return null;
    }
  }

  static Future<String?> embedMetadata({
    required String flacPath,
    String? coverPath,
    Map<String, String>? metadata,
  }) async {
    final tempDir = await getTemporaryDirectory();
    final uniqueId = DateTime.now().millisecondsSinceEpoch;
    final tempOutput = '${tempDir.path}/temp_embed_$uniqueId.flac';
    
    final StringBuffer cmdBuffer = StringBuffer();
    cmdBuffer.write('-i "$flacPath" ');
    
    if (coverPath != null) {
      cmdBuffer.write('-i "$coverPath" ');
    }
    
    cmdBuffer.write('-map 0:a ');
    
    if (coverPath != null) {
      cmdBuffer.write('-map 1:0 ');
      cmdBuffer.write('-c:v copy ');
      cmdBuffer.write('-disposition:v attached_pic ');
      cmdBuffer.write('-metadata:s:v title="Album cover" ');
      cmdBuffer.write('-metadata:s:v comment="Cover (front)" ');
    }
    
    cmdBuffer.write('-c:a copy ');
    
    if (metadata != null) {
      metadata.forEach((key, value) {
        final sanitizedValue = value.replaceAll('"', '\\"');
        cmdBuffer.write('-metadata $key="$sanitizedValue" ');
      });
    }
    
    cmdBuffer.write('"$tempOutput" -y');
    
    final command = cmdBuffer.toString();
    _log.d('Executing FFmpeg command: $command');

    final result = await _execute(command);

    if (result.success) {
      try {
        final tempFile = File(tempOutput);
        final originalFile = File(flacPath);
        
        if (await tempFile.exists()) {
             if (await originalFile.exists()) {
               await originalFile.delete();
             }
             await tempFile.copy(flacPath);
             await tempFile.delete();
             
             return flacPath;
        } else {
             _log.e('Temp output file not found: $tempOutput');
             return null;
        }

      } catch (e) {
        _log.e('Failed to replace file after metadata embed: $e');
        return null;
      }
    }

    try {
      final tempFile = File(tempOutput);
      if (await tempFile.exists()) {
        await tempFile.delete();
      }
    } catch (e) {
      _log.w('Failed to cleanup temp file: $e');
    }

    _log.e('Metadata/Cover embed failed: ${result.output}');
    return null;
  }

  static Future<String?> embedMetadataToMp3({
    required String mp3Path,
    String? coverPath,
    Map<String, String>? metadata,
  }) async {
    final tempDir = await getTemporaryDirectory();
    final uniqueId = DateTime.now().millisecondsSinceEpoch;
    final tempOutput = '${tempDir.path}/temp_embed_$uniqueId.mp3';
    
    final StringBuffer cmdBuffer = StringBuffer();
    cmdBuffer.write('-i "$mp3Path" ');
    
    if (coverPath != null) {
      cmdBuffer.write('-i "$coverPath" ');
    }
    
    cmdBuffer.write('-map 0:a ');
    
    if (coverPath != null) {
      cmdBuffer.write('-map 1:0 ');
      cmdBuffer.write('-c:v:0 copy ');
      cmdBuffer.write('-id3v2_version 3 ');
      cmdBuffer.write('-metadata:s:v title="Album cover" ');
      cmdBuffer.write('-metadata:s:v comment="Cover (front)" ');
    }
    
    cmdBuffer.write('-c:a copy ');
    
    if (metadata != null) {
      final id3Metadata = _convertToId3Tags(metadata);
      id3Metadata.forEach((key, value) {
        final sanitizedValue = value.replaceAll('"', '\\"');
        cmdBuffer.write('-metadata $key="$sanitizedValue" ');
      });
    }
    
    cmdBuffer.write('-id3v2_version 3 "$tempOutput" -y');
    
    final command = cmdBuffer.toString();
    _log.d('Executing FFmpeg MP3 embed command: $command');

    final result = await _execute(command);

    if (result.success) {
      try {
        final tempFile = File(tempOutput);
        final originalFile = File(mp3Path);
        
        if (await tempFile.exists()) {
             if (await originalFile.exists()) {
               await originalFile.delete();
             }
             await tempFile.copy(mp3Path);
             await tempFile.delete();
             
             _log.d('MP3 metadata embedded successfully');
             return mp3Path;
        } else {
             _log.e('Temp MP3 output file not found: $tempOutput');
             return null;
        }

      } catch (e) {
        _log.e('Failed to replace MP3 file after metadata embed: $e');
        return null;
      }
    }

    try {
      final tempFile = File(tempOutput);
      if (await tempFile.exists()) {
        await tempFile.delete();
      }
    } catch (e) {
      _log.w('Failed to cleanup temp MP3 file: $e');
    }

    _log.e('MP3 Metadata/Cover embed failed: ${result.output}');
    return null;
  }

  /// Embed metadata to Opus file
  /// Uses METADATA_BLOCK_PICTURE tag for cover art (OGG/Vorbis standard)
  static Future<String?> embedMetadataToOpus({
    required String opusPath,
    String? coverPath,
    Map<String, String>? metadata,
  }) async {
    final tempDir = await getTemporaryDirectory();
    final uniqueId = DateTime.now().millisecondsSinceEpoch;
    final tempOutput = '${tempDir.path}/temp_embed_$uniqueId.opus';
    
    final StringBuffer cmdBuffer = StringBuffer();
    cmdBuffer.write('-i "$opusPath" ');
    cmdBuffer.write('-map 0:a ');
    cmdBuffer.write('-c:a copy ');
    
    // Embed metadata tags (Vorbis comments)
    if (metadata != null) {
      metadata.forEach((key, value) {
        final sanitizedValue = value.replaceAll('"', '\\"');
        cmdBuffer.write('-metadata $key="$sanitizedValue" ');
      });
    }
    
    // Embed cover art using METADATA_BLOCK_PICTURE
    if (coverPath != null) {
      try {
        final pictureBlock = await _createMetadataBlockPicture(coverPath);
        if (pictureBlock != null) {
          // Escape special characters for shell
          final escapedBlock = pictureBlock.replaceAll('"', '\\"');
          cmdBuffer.write('-metadata METADATA_BLOCK_PICTURE="$escapedBlock" ');
          _log.d('Created METADATA_BLOCK_PICTURE for Opus (${pictureBlock.length} chars)');
        } else {
          _log.w('Failed to create METADATA_BLOCK_PICTURE, skipping cover');
        }
      } catch (e) {
        _log.e('Error creating METADATA_BLOCK_PICTURE: $e');
      }
    }
    
    cmdBuffer.write('"$tempOutput" -y');
    
    final command = cmdBuffer.toString();
    _log.d('Executing FFmpeg Opus embed command');

    final result = await _execute(command);

    if (result.success) {
      try {
        final tempFile = File(tempOutput);
        final originalFile = File(opusPath);
        
        if (await tempFile.exists()) {
             if (await originalFile.exists()) {
               await originalFile.delete();
             }
             await tempFile.copy(opusPath);
             await tempFile.delete();
             
             _log.d('Opus metadata embedded successfully');
             return opusPath;
        } else {
             _log.e('Temp Opus output file not found: $tempOutput');
             return null;
        }

      } catch (e) {
        _log.e('Failed to replace Opus file after metadata embed: $e');
        return null;
      }
    }

    try {
      final tempFile = File(tempOutput);
      if (await tempFile.exists()) {
        await tempFile.delete();
      }
    } catch (e) {
      _log.w('Failed to cleanup temp Opus file: $e');
    }

    _log.e('Opus Metadata embed failed: ${result.output}');
    return null;
  }

  /// Create METADATA_BLOCK_PICTURE base64 string for OGG/Opus cover art
  /// Format follows FLAC picture block specification:
  /// - 4 bytes: picture type (3 = front cover)
  /// - 4 bytes: MIME type length
  /// - n bytes: MIME type string
  /// - 4 bytes: description length
  /// - n bytes: description string
  /// - 4 bytes: width
  /// - 4 bytes: height
  /// - 4 bytes: color depth
  /// - 4 bytes: colors used (0 for non-indexed)
  /// - 4 bytes: picture data length
  /// - n bytes: picture data
  static Future<String?> _createMetadataBlockPicture(String imagePath) async {
    try {
      final file = File(imagePath);
      if (!await file.exists()) {
        _log.e('Cover image not found: $imagePath');
        return null;
      }
      
      final imageData = await file.readAsBytes();
      
      // Detect MIME type from file extension or magic bytes
      String mimeType;
      if (imagePath.toLowerCase().endsWith('.png')) {
        mimeType = 'image/png';
      } else if (imagePath.toLowerCase().endsWith('.jpg') || 
                 imagePath.toLowerCase().endsWith('.jpeg')) {
        mimeType = 'image/jpeg';
      } else {
        // Check magic bytes
        if (imageData.length >= 8 && 
            imageData[0] == 0x89 && imageData[1] == 0x50 && 
            imageData[2] == 0x4E && imageData[3] == 0x47) {
          mimeType = 'image/png';
        } else if (imageData.length >= 2 && 
                   imageData[0] == 0xFF && imageData[1] == 0xD8) {
          mimeType = 'image/jpeg';
        } else {
          mimeType = 'image/jpeg'; // Default to JPEG
        }
      }
      
      final mimeBytes = utf8.encode(mimeType);
      const description = ''; // Empty description
      final descBytes = utf8.encode(description);
      
      // Build the FLAC picture block
      // Total size: 4 + 4 + mimeLen + 4 + descLen + 4 + 4 + 4 + 4 + 4 + imageLen
      final blockSize = 4 + 4 + mimeBytes.length + 4 + descBytes.length + 
                        4 + 4 + 4 + 4 + 4 + imageData.length;
      
      final buffer = ByteData(blockSize);
      var offset = 0;
      
      // Picture type: 3 = Front cover
      buffer.setUint32(offset, 3, Endian.big);
      offset += 4;
      
      // MIME type length
      buffer.setUint32(offset, mimeBytes.length, Endian.big);
      offset += 4;
      
      // MIME type string
      final blockBytes = Uint8List(blockSize);
      blockBytes.setRange(0, offset, buffer.buffer.asUint8List());
      blockBytes.setRange(offset, offset + mimeBytes.length, mimeBytes);
      offset += mimeBytes.length;
      
      // Description length
      final tempBuffer = ByteData(4);
      tempBuffer.setUint32(0, descBytes.length, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Description string
      blockBytes.setRange(offset, offset + descBytes.length, descBytes);
      offset += descBytes.length;
      
      // Width (0 = unknown)
      tempBuffer.setUint32(0, 0, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Height (0 = unknown)
      tempBuffer.setUint32(0, 0, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Color depth (0 = unknown)
      tempBuffer.setUint32(0, 0, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Colors used (0 for non-indexed)
      tempBuffer.setUint32(0, 0, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Picture data length
      tempBuffer.setUint32(0, imageData.length, Endian.big);
      blockBytes.setRange(offset, offset + 4, tempBuffer.buffer.asUint8List());
      offset += 4;
      
      // Picture data
      blockBytes.setRange(offset, offset + imageData.length, imageData);
      
      // Base64 encode the entire block
      final base64String = base64Encode(blockBytes);
      
      return base64String;
    } catch (e) {
      _log.e('Error creating METADATA_BLOCK_PICTURE: $e');
      return null;
    }
  }

  static Map<String, String> _convertToId3Tags(Map<String, String> vorbisMetadata) {
    final id3Map = <String, String>{};
    
    for (final entry in vorbisMetadata.entries) {
      final key = entry.key.toUpperCase();
      final value = entry.value;
      
      // Map Vorbis comments to ID3v2 frame names
      switch (key) {
        case 'TITLE':
          id3Map['title'] = value;
          break;
        case 'ARTIST':
          id3Map['artist'] = value;
          break;
        case 'ALBUM':
          id3Map['album'] = value;
          break;
        case 'ALBUMARTIST':
          id3Map['album_artist'] = value;
          break;
        case 'TRACKNUMBER':
        case 'TRACK':
          id3Map['track'] = value;
          break;
        case 'DISCNUMBER':
        case 'DISC':
          id3Map['disc'] = value;
          break;
        case 'DATE':
        case 'YEAR':
          id3Map['date'] = value;
          break;
        case 'ISRC':
          id3Map['TSRC'] = value;
          break;
        case 'LYRICS':
        case 'UNSYNCEDLYRICS':
          id3Map['lyrics'] = value;
          break;
        default:
          // Pass through other tags as-is
          id3Map[key.toLowerCase()] = value;
      }
    }
    
    return id3Map;
  }
}

class FFmpegResult {
  final bool success;
  final int returnCode;
  final String output;

  FFmpegResult({
    required this.success,
    required this.returnCode,
    required this.output,
  });
}
