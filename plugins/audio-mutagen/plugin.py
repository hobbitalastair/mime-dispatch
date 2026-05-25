#!/usr/bin/env python3
"""Audio metadata write plugin using mutagen.

Provides metadata-add and metadata-delete for audio files
(audio/mpeg, audio/ogg, audio/x-vorbis+ogg, audio/flac).
Supports multiple values for genre and other tags.

Uses mutagen's Easy interface for a consistent dict-like API
across formats. Handles MP3 comment frames specially via raw ID3
since EasyID3 does not expose a 'comment' key.
"""

import argparse
import os
import sys

MIMETYPES = [
    "audio/mpeg",
    "audio/ogg",
    "audio/x-vorbis+ogg",
    "audio/flac",
    "audio/mp4",
]

COMMANDS = ["metadata-add", "metadata-delete"]

KEY_MAP = {
    "title": "title",
    "album": "album",
    "artist": "artist",
    "album_artist": "albumartist",
    "composer": "composer",
    "genre": "genre",
    "year": "date",
    "comment": "comment",
}


def main():
    parser = argparse.ArgumentParser(
        prog=os.path.basename(sys.argv[0]),
        usage="%(prog)s <file> <key> <value>",
        add_help=False,
    )
    parser.add_argument("--capabilities", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("file", nargs="?")
    parser.add_argument("key", nargs="?")
    parser.add_argument("value", nargs="?")

    args = parser.parse_args()

    if args.capabilities:
        print_capabilities()
        return

    if not all([args.file, args.key, args.value]):
        usage()
        sys.exit(1)

    command = os.path.basename(sys.argv[0])

    if command == "metadata-add":
        add_metadata(args.file, args.key, args.value)
    elif command == "metadata-delete":
        delete_metadata(args.file, args.key, args.value)
    else:
        usage()
        sys.exit(1)


def print_capabilities():
    sys.stdout.write("mimetypes:\n")
    for mt in MIMETYPES:
        sys.stdout.write(f"    - {mt}\n")
    sys.stdout.write("commands:\n")
    for cmd in COMMANDS:
        sys.stdout.write(f"    - {cmd}\n")


def usage():
    name = os.path.basename(sys.argv[0])
    print(f"Usage: {name} <file> <key> <value>", file=sys.stderr)
    print(f"Supported keys: {', '.join(sorted(KEY_MAP.keys()))}", file=sys.stderr)
    sys.exit(1)


def die(message):
    print(f"Error: {message}", file=sys.stderr)
    sys.exit(1)


def load_audio(filepath):
    from mutagen import File

    if not os.path.exists(filepath):
        die(f"file not found: {filepath}")

    audio = File(filepath, easy=True)
    if audio is None:
        die(f"unsupported or unreadable file: {filepath}")

    return audio


def map_key(key):
    mapped = KEY_MAP.get(key)
    if mapped is None:
        return None
    return mapped


def is_mp3(audio):
    from mutagen.mp3 import MP3

    return isinstance(audio, MP3)


# ---- MP3 comment helpers (raw ID3, since EasyID3 lacks a 'comment' key) ----


def _add_comment_mp3(filepath, value):
    from mutagen.id3 import ID3, COMM, ID3NoHeaderError

    try:
        tags = ID3(filepath)
    except ID3NoHeaderError:
        tags = ID3()

    for key in list(tags.keys()):
        if key.startswith("COMM::"):
            frame = tags[key]
            if frame.desc == "" and value not in frame.text:
                frame.text = list(frame.text) + [value]
                tags.save()
                return

    tags["COMM::eng"] = COMM(encoding=3, lang="eng", desc="", text=[value])
    tags.save()


def _delete_comment_mp3(filepath, value):
    from mutagen.id3 import ID3, ID3NoHeaderError

    try:
        tags = ID3(filepath)
    except ID3NoHeaderError:
        return

    for key in list(tags.keys()):
        if key.startswith("COMM::"):
            frame = tags[key]
            if frame.desc == "" and value in frame.text:
                filtered = [v for v in frame.text if v != value]
                if filtered:
                    frame.text = filtered
                else:
                    del tags[key]
                tags.save()
                return


# ---- Public API ----


def add_metadata(filepath, key, value):
    k = map_key(key)
    if k is None:
        return  # unknown key, pass through to xattr

    audio = load_audio(filepath)

    if is_mp3(audio) and k == "comment":
        _add_comment_mp3(filepath, value)
        return

    values = audio.get(k, [])
    if value not in values:
        values.append(value)

    audio[k] = values
    audio.save()


def delete_metadata(filepath, key, value):
    k = map_key(key)
    if k is None:
        return  # unknown key, pass through to xattr

    audio = load_audio(filepath)

    if is_mp3(audio) and k == "comment":
        _delete_comment_mp3(filepath, value)
        return

    if k not in audio:
        return

    values = audio[k]
    filtered = [v for v in values if v != value]

    if filtered:
        audio[k] = filtered
    else:
        del audio[k]

    audio.save()


if __name__ == "__main__":
    main()
