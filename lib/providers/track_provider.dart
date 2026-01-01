import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:spotiflac_android/models/track.dart';
import 'package:spotiflac_android/services/platform_bridge.dart';

class TrackState {
  final List<Track> tracks;
  final bool isLoading;
  final String? error;
  final String? albumName;
  final String? playlistName;
  final String? coverUrl;

  const TrackState({
    this.tracks = const [],
    this.isLoading = false,
    this.error,
    this.albumName,
    this.playlistName,
    this.coverUrl,
  });

  TrackState copyWith({
    List<Track>? tracks,
    bool? isLoading,
    String? error,
    String? albumName,
    String? playlistName,
    String? coverUrl,
  }) {
    return TrackState(
      tracks: tracks ?? this.tracks,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      albumName: albumName ?? this.albumName,
      playlistName: playlistName ?? this.playlistName,
      coverUrl: coverUrl ?? this.coverUrl,
    );
  }
}

class TrackNotifier extends Notifier<TrackState> {
  @override
  TrackState build() {
    return const TrackState();
  }

  Future<void> fetchFromUrl(String url) async {
    state = state.copyWith(isLoading: true, error: null);

    try {
      final parsed = await PlatformBridge.parseSpotifyUrl(url);
      final type = parsed['type'] as String;

      final metadata = await PlatformBridge.getSpotifyMetadata(url);

      if (type == 'track') {
        final trackData = metadata['track'] as Map<String, dynamic>;
        final track = _parseTrack(trackData);
        state = state.copyWith(
          tracks: [track],
          isLoading: false,
          albumName: null,
          playlistName: null,
          coverUrl: track.coverUrl,
        );
      } else if (type == 'album') {
        final albumInfo = metadata['album_info'] as Map<String, dynamic>;
        final trackList = metadata['track_list'] as List<dynamic>;
        final tracks = trackList.map((t) => _parseTrack(t as Map<String, dynamic>)).toList();
        state = state.copyWith(
          tracks: tracks,
          isLoading: false,
          albumName: albumInfo['name'] as String?,
          playlistName: null,
          coverUrl: albumInfo['images'] as String?,
        );
      } else if (type == 'playlist') {
        final playlistInfo = metadata['playlist_info'] as Map<String, dynamic>;
        final trackList = metadata['track_list'] as List<dynamic>;
        final tracks = trackList.map((t) => _parseTrack(t as Map<String, dynamic>)).toList();
        final owner = playlistInfo['owner'] as Map<String, dynamic>?;
        state = state.copyWith(
          tracks: tracks,
          isLoading: false,
          albumName: null,
          playlistName: owner?['name'] as String?,
          coverUrl: owner?['images'] as String?,
        );
      }
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> search(String query) async {
    state = state.copyWith(isLoading: true, error: null);

    try {
      final results = await PlatformBridge.searchSpotify(query, limit: 20);
      final trackList = results['tracks'] as List<dynamic>? ?? [];
      final tracks = trackList.map((t) => _parseSearchTrack(t as Map<String, dynamic>)).toList();
      state = state.copyWith(
        tracks: tracks,
        isLoading: false,
        albumName: null,
        playlistName: null,
      );
    } catch (e) {
      state = state.copyWith(isLoading: false, error: e.toString());
    }
  }

  Future<void> checkAvailability(int index) async {
    if (index < 0 || index >= state.tracks.length) return;

    final track = state.tracks[index];
    if (track.isrc == null || track.isrc!.isEmpty) return;

    try {
      final availability = await PlatformBridge.checkAvailability(track.id, track.isrc!);
      final updatedTrack = Track(
        id: track.id,
        name: track.name,
        artistName: track.artistName,
        albumName: track.albumName,
        albumArtist: track.albumArtist,
        coverUrl: track.coverUrl,
        isrc: track.isrc,
        duration: track.duration,
        trackNumber: track.trackNumber,
        discNumber: track.discNumber,
        releaseDate: track.releaseDate,
        availability: ServiceAvailability(
          tidal: availability['tidal'] as bool? ?? false,
          qobuz: availability['qobuz'] as bool? ?? false,
          amazon: availability['amazon'] as bool? ?? false,
          tidalUrl: availability['tidal_url'] as String?,
          qobuzUrl: availability['qobuz_url'] as String?,
          amazonUrl: availability['amazon_url'] as String?,
        ),
      );

      final tracks = List<Track>.from(state.tracks);
      tracks[index] = updatedTrack;
      state = state.copyWith(tracks: tracks);
    } catch (e) {
      // Silently fail availability check
    }
  }

  void clear() {
    state = const TrackState();
  }

  Track _parseTrack(Map<String, dynamic> data) {
    return Track(
      id: data['spotify_id'] as String? ?? '',
      name: data['name'] as String? ?? '',
      artistName: data['artists'] as String? ?? '',
      albumName: data['album_name'] as String? ?? '',
      albumArtist: data['album_artist'] as String?,
      coverUrl: data['images'] as String?,
      isrc: data['isrc'] as String?,
      duration: data['duration_ms'] as int? ?? 0,
      trackNumber: data['track_number'] as int?,
      discNumber: data['disc_number'] as int?,
      releaseDate: data['release_date'] as String?,
    );
  }

  Track _parseSearchTrack(Map<String, dynamic> data) {
    return Track(
      id: data['spotify_id'] as String? ?? '',
      name: data['name'] as String? ?? '',
      artistName: data['artists'] as String? ?? '',
      albumName: data['album_name'] as String? ?? '',
      albumArtist: data['album_artist'] as String?,
      coverUrl: data['images'] as String?,
      isrc: data['isrc'] as String?,
      duration: data['duration_ms'] as int? ?? 0,
      trackNumber: data['track_number'] as int?,
      discNumber: data['disc_number'] as int?,
      releaseDate: data['release_date'] as String?,
    );
  }
}

final trackProvider = NotifierProvider<TrackNotifier, TrackState>(
  TrackNotifier.new,
);
