import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:spotiflac_android/models/download_item.dart';
import 'package:spotiflac_android/providers/download_queue_provider.dart';

class QueueTab extends ConsumerWidget {
  const QueueTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final queueState = ref.watch(downloadQueueProvider);
    final colorScheme = Theme.of(context).colorScheme;

    return Column(
      children: [
        // Header with actions
        if (queueState.items.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '${queueState.items.length} items',
                  style: Theme.of(context).textTheme.titleSmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                Row(
                  children: [
                    TextButton.icon(
                      onPressed: () => ref.read(downloadQueueProvider.notifier).clearCompleted(),
                      icon: const Icon(Icons.done_all, size: 18),
                      label: const Text('Clear done'),
                    ),
                    TextButton.icon(
                      onPressed: () => _showClearAllDialog(context, ref),
                      icon: Icon(Icons.clear_all, size: 18, color: colorScheme.error),
                      label: Text('Clear all', style: TextStyle(color: colorScheme.error)),
                    ),
                  ],
                ),
              ],
            ),
          ),
        
        // Queue list
        Expanded(
          child: queueState.items.isEmpty
              ? _buildEmptyState(context, colorScheme)
              : ListView.builder(
                  itemCount: queueState.items.length,
                  itemBuilder: (context, index) => _buildQueueItem(context, ref, queueState.items[index], colorScheme),
                ),
        ),
      ],
    );
  }

  Widget _buildEmptyState(BuildContext context, ColorScheme colorScheme) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.queue_music, 
            size: 64, 
            color: colorScheme.onSurfaceVariant,
          ),
          const SizedBox(height: 16),
          Text(
            'No downloads in queue',
            style: Theme.of(context).textTheme.bodyLarge?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Add tracks from the Home tab',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: colorScheme.onSurfaceVariant.withValues(alpha: 0.7),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildQueueItem(BuildContext context, WidgetRef ref, DownloadItem item, ColorScheme colorScheme) {
    return ListTile(
      leading: item.track.coverUrl != null
          ? ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: CachedNetworkImage(
                imageUrl: item.track.coverUrl!,
                width: 48,
                height: 48,
                fit: BoxFit.cover,
              ),
            )
          : Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(Icons.music_note, color: colorScheme.onSurfaceVariant),
            ),
      title: Text(item.track.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            item.track.artistName, 
            maxLines: 1, 
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: colorScheme.onSurfaceVariant),
          ),
          if (item.status == DownloadStatus.downloading) ...[
            const SizedBox(height: 4),
            Row(
              children: [
                Expanded(
                  child: LinearProgressIndicator(
                    value: item.progress > 0 ? item.progress : null,
                    backgroundColor: colorScheme.surfaceContainerHighest,
                    color: colorScheme.primary,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  '${(item.progress * 100).toStringAsFixed(0)}%',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
      trailing: _buildStatusIcon(context, item, colorScheme),
      onTap: item.status == DownloadStatus.queued
          ? () => ref.read(downloadQueueProvider.notifier).cancelItem(item.id)
          : null,
    );
  }

  Widget _buildStatusIcon(BuildContext context, DownloadItem item, ColorScheme colorScheme) {
    switch (item.status) {
      case DownloadStatus.queued:
        return Icon(Icons.hourglass_empty, color: colorScheme.onSurfaceVariant);
      case DownloadStatus.downloading:
        return SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(
            value: item.progress,
            strokeWidth: 2,
            color: colorScheme.primary,
          ),
        );
      case DownloadStatus.completed:
        return Icon(Icons.check_circle, color: colorScheme.primary);
      case DownloadStatus.failed:
        return IconButton(
          icon: Icon(Icons.error, color: colorScheme.error),
          onPressed: () => _showErrorDialog(context, item, colorScheme),
          tooltip: 'Tap to see error details',
        );
      case DownloadStatus.skipped:
        return Icon(Icons.skip_next, color: colorScheme.primary);
    }
  }

  void _showErrorDialog(BuildContext context, DownloadItem item, ColorScheme colorScheme) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Row(
          children: [
            Icon(Icons.error, color: colorScheme.error),
            const SizedBox(width: 8),
            const Text('Download Failed'),
          ],
        ),
        content: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Track: ${item.track.name}', style: const TextStyle(fontWeight: FontWeight.bold)),
              Text('Artist: ${item.track.artistName}'),
              const SizedBox(height: 16),
              const Text('Error:', style: TextStyle(fontWeight: FontWeight.bold)),
              const SizedBox(height: 4),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: colorScheme.errorContainer,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  item.error ?? 'Unknown error',
                  style: TextStyle(
                    fontFamily: 'monospace', 
                    fontSize: 12,
                    color: colorScheme.onErrorContainer,
                  ),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  void _showClearAllDialog(BuildContext context, WidgetRef ref) {
    final colorScheme = Theme.of(context).colorScheme;
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Clear All'),
        content: const Text('Are you sure you want to clear all downloads?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              ref.read(downloadQueueProvider.notifier).clearAll();
              Navigator.pop(context);
            },
            child: Text('Clear', style: TextStyle(color: colorScheme.error)),
          ),
        ],
      ),
    );
  }
}
