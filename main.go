package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ffProbeOutput struct {
	Video   stream
	Audio   stream
	Streams []stream `json:"streams"`
}

type stream struct {
	CodecName    string `json:"codec_long_name"`
	CodecType    string `json:"codec_type"`
	PixelFormat  string `json:"pix_fmt"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	TimeBase     string `json:"time_base"`
	SampleRate   string `json:"sample_rate"`
	BitDepth     string `json:"bits_per_raw_sample"`
}

type fileArgs struct {
	inputDir   string
	videoFiles []string
	outputFile string
}

func main() {
	fileArgs, err := getArgs()
	if err != nil {
		log.Fatalf("failed to get args: %v", err)
	}

	videoPaths, err := getVideoPaths(fileArgs)
	if err != nil {
		log.Fatalf("failed to get video paths: %v", err)
	}

	if err := checkSimilarity(videoPaths); err != nil {
		log.Fatalf("similarity check failed: %v", err)
	}

	if err := concatenateVideos(fileArgs, getInputList(videoPaths)); err != nil {
		log.Fatalf("failed to concatenate: %v", err)
	}
}

func checkSimilarity(videoPaths []string) error {
	var firstStreamVideo stream
	var firstStreamAudio stream
	var firstExtension string

	for i, videoPath := range videoPaths {
		extension := filepath.Ext(videoPath)
		if i == 0 {
			firstExtension = extension
		}
		if extension != firstExtension {
			return fmt.Errorf("got mismatched video extensions %v and %v", extension, firstExtension)
		}

		ffProbeOutput, err := execFfprobeCommand(videoPath)
		if err != nil {
			return fmt.Errorf("encountered error running ffprobe for %v - %v", videoPath, err)
		}

		if i == 0 {
			firstStreamVideo = ffProbeOutput.Video
			firstStreamAudio = ffProbeOutput.Audio
		}
		if firstStreamVideo != ffProbeOutput.Video {
			return fmt.Errorf("file %v did not match first video probe. %v vs %v",
				videoPath, ffProbeOutput.Video, firstStreamVideo)
		}
		if firstStreamAudio != ffProbeOutput.Audio {
			return fmt.Errorf("file %v did not match first audio probe. %v vs %v",
				videoPath, ffProbeOutput.Audio, firstStreamAudio)
		}
	}
	return nil
}

func getInputList(videoPaths []string) string {
	var inputList string
	for _, videoPath := range videoPaths {
		inputList += fmt.Sprintf("file '%s'\n", videoPath)
	}
	return inputList
}

func getVideoPaths(fileArgs *fileArgs) ([]string, error) {
	videoPaths := []string{}
	for _, videoFile := range fileArgs.videoFiles {
		videoPath, err := getVideoPath(videoFile, fileArgs.inputDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get video path: %v", err)
		}
		videoPaths = append(videoPaths, videoPath)
	}
	return videoPaths, nil
}

func concatenateVideos(fileArgs *fileArgs, inputList string) error {
	tempFile, err := createTempFile(fileArgs.inputDir, inputList)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}

	if output, err := execFfmpegCommand(tempFile, fileArgs.outputFile); err != nil {
		return fmt.Errorf("ffmpeg failed: %v\nOutput: %s", err, output)
	}
	if err := deleteFile(tempFile); err != nil {
		return fmt.Errorf("failed to delete temp file: %v", err)
	}
	return nil
}

func getArgs() (*fileArgs, error) {
	args := os.Args
	if len(args) < 4 {
		return nil, fmt.Errorf("need at least 3 args for input directory, "+
			"video inputs (comma separated) and output file name, got %v", len(args))
	}
	inputDir := args[1]
	videoInputs := strings.Split(args[2], ",")
	if len(videoInputs) < 2 {
		return nil, fmt.Errorf("expected at least 2 input videos, got %v", len(videoInputs))
	}
	outputFile := filepath.Join(inputDir, args[3])
	return &fileArgs{inputDir: inputDir, videoFiles: videoInputs, outputFile: outputFile}, nil
}

func deleteFile(file string) error {
	return os.Remove(file)
}

func createTempFile(inputDir, inputList string) (string, error) {
	tempFile := filepath.Join(inputDir, fmt.Sprintf("tmp%d.txt", time.Now().Unix()))
	if err := os.WriteFile(tempFile, []byte(inputList), 0644); err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	return tempFile, nil
}

func getVideoPath(file string, inputDir string) (string, error) {
	videoPath := filepath.Join(inputDir, file)
	fileInfo, err := os.Stat(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to find video %v - %v", videoPath, err)
	}
	if fileInfo.IsDir() {
		return "", fmt.Errorf("provided video file %v is a directory", videoPath)
	}
	return videoPath, nil
}

func execFfmpegCommand(tempFile string, outputFile string) ([]byte, error) {
	cmd := exec.Command(
		// TODO - do we want to use y flag here?
		"ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", tempFile, "-c", "copy", outputFile)

	log.Println("Running the following command:", strings.Join(cmd.Args, " "))

	return cmd.CombinedOutput()
}

func execFfprobeCommand(input string) (*ffProbeOutput, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format",
		"json", "-show_streams", input)
	log.Println("Running the following command:", strings.Join(cmd.Args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run ffprobe: %v - %s", err, output)
	}

	var ffprobeOutput *ffProbeOutput
	err = json.Unmarshal(output, &ffprobeOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	var videoStream, audioStream *stream
	for _, s := range ffprobeOutput.Streams {
		if s.CodecType == "video" {
			videoStream = &s
		}
		if s.CodecType == "audio" {
			audioStream = &s
		}
	}

	if videoStream != nil && audioStream != nil {
		ffprobeOutput.Video = *videoStream
		ffprobeOutput.Audio = *audioStream
	} else {
		return nil, fmt.Errorf("failed to find video and audio streams")
	}

	return ffprobeOutput, nil
}
