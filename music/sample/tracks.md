Sample Music Library
====================

Royalty-free tracks from Pixabay (pixabay.com/music).
Download each track and save to the matching mood folder.


Focus (ambient instrumental)
-----------------------------
Save to: music/sample/focus/

Ethereal Ambient Atmosphere         0:55  https://pixabay.com/music/ambient-ethereal-ambient-atmosphere-210335/
An Eye for Details                  1:07  https://pixabay.com/music/ambient-an-eye-for-details-384183/
Ambient Background Music            1:19  https://pixabay.com/music/ambient-ambient-background-music-for-focus-and-relaxation-342438/
Minimal Ambient Background          2:44  https://pixabay.com/music/ambient-minimal-ambient-background-472197/


Calm (lofi / relaxing)
-----------------------
Save to: music/sample/calm/

Lofi Relaxing Chill Music           1:25  https://pixabay.com/music/beats-lofi-relaxing-chill-music-295834/
Dreamy Lofi Nostalgic Background    2:04  https://pixabay.com/music/lofi-dreamy-lofi-nostalgic-background-469629/
Sentimental Jazzy Love              1:40  https://pixabay.com/music/lofi-lo-fi-music-loop-sentimental-jazzy-love-473154/           [vocals]
Lofi Study Calm Peaceful Chill Hop  2:27  https://pixabay.com/music/beats-lofi-study-calm-peaceful-chill-hop-112191/               [vocals]


Energize (upbeat / electronic)
-------------------------------
Save to: music/sample/energize/

Stylish Deep Electronic             1:36  https://pixabay.com/music/future-bass-stylish-deep-electronic-262632/
Upbeat Electronic Beat              1:45  https://pixabay.com/music/upbeat-upbeat-electronic-beat-347211/
Upbeat Music 2                      1:26  https://pixabay.com/music/upbeat-upbeat-music-2-405217/                                  [vocals]
Running Night                       1:52  https://pixabay.com/music/funk-running-night-393139/                                     [vocals]


Late Night (downtempo / atmospheric)
-------------------------------------
Save to: music/sample/late_night/

Shadows in Motion Short 2           0:55  https://pixabay.com/music/beats-shadows-in-motion-short-2-367703/
Shadows in Motion Short 1           1:39  https://pixabay.com/music/beats-shadows-in-motion-short-1-367702/
LoFi Chill Downtempo                1:10  https://pixabay.com/music/beats-lofi-chill-downtempo-225107/                             [vocals]
Marmalade                           1:47  https://pixabay.com/music/beats-marmalade-411291/                                        [vocals] [lyrics]


Notes
------
- Focus tracks are always instrumental (no .txt file).
- To mark a track as vocal, place a .txt file next to the .mp3:
    marmalade-411291.mp3 + marmalade-411291.txt → vocal
    ocean-waves-12345.mp3 (no .txt)             → instrumental
- The .txt can contain lyrics or be empty. Both work.
- Lyrics format: plain text, one line per lyric, blank lines between
  stanzas. See late_night/marmalade-411291.txt for an example.


After downloading, import and run:

make db-init
make import-batch ARGS="music/sample/focus"
make import-batch ARGS="music/sample/calm"
make import-batch ARGS="music/sample/energize"
make import-batch ARGS="music/sample/late_night"
make run
