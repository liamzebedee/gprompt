download(playlists, dir):
    Keep track of my youtube playlists.

    Download all of the songs from youtube by using these sites.
    Download dir: [dir]
    
    Download sites:
    - https://app.ytmp3.as/

    Playlists:
    [playlists]

song-normalise:
    You have .mp3's in this directory.
    Your job is to normalise and annotate them:

    - Get the thumbnails from youtube too.
    - Add the metadata for song title, artist, album. 
    - Remove any extraneous stuff (like something.com as distributor).


agent(loop(download-songs))

; starts an agent for song normalisation in background
agent(loop(song-normalise))