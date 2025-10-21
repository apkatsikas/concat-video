# Video Concatenation Tool

A Go application that concatenates multiple video files into a single output file using `ffmpeg` and `ffprobe`. The tool checks if video files have compatible properties and concatenates them without re-encoding if possible.

## Requirements

- **ffmpeg** and **ffprobe** installed and available in your system's `PATH`.

## Usage

`concat-video /path/to/folder video001.mp4,video002.mp4,video003.mp4 combined.mp4`
