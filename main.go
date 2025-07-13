package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
)

func main() {
	run()
}

var (
	contextSize = flag.Int("tcs", 2, "context size")
	batchSize   = flag.Int("tbs", 1, "batch size")
	ollamaModel = flag.String("tom", "", "ollama model name")
	message     = flag.String("tm", "", "a message that describe the content of transcribed text to make better translation")
	filePath    = flag.String("tf", "", "file path")
)

func run() {
	flag.Parse()
	args := flag.Args()

	ffmpegCmd := exec.Command("ffmpeg", "-i", *filePath, "-vn", "-acodec", "pcm_s16le", "-ar", "44100", "-ac", "1", "output.wav")
	ffmpegCmd.Stderr = os.Stderr
	ffmpegCmd.Stdout = os.Stdout
	err := ffmpegCmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	translater := NewTranslate(*message, *ollamaModel, *contextSize, *batchSize)

	whisperCmd := exec.Command("whisper-cli", append(args, "-f", "output.wav")...)

	whisperOut, err := whisperCmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer whisperOut.Close()

	var whisperOutPut []byte
	outBuf := make([]byte, 1024)

	//todo
	done := make(chan struct{})

	go func() {
		reader := bufio.NewReader(whisperOut)
		for {
			n, err := reader.Read(outBuf)
			if n > 0 {
				whisperOutPut = append(whisperOutPut, outBuf[:n]...)
				os.Stdout.Write(outBuf[:n])
			}
			if err != nil {
				if err != io.EOF {
					fmt.Fprintln(os.Stderr, "read error:", err)
				}
				break
			}
		}
		close(done)
	}()

	whisperErr, err := whisperCmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer whisperErr.Close()

	go func() {
		scanner := bufio.NewScanner(whisperErr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "%s\n", scanner.Text())
		}
	}()

	err = whisperCmd.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(bytes.NewReader(whisperOutPut))

	translater.process(scanner)

	srtFileName := path.Base(*filePath) + ".srt"

	WriteSrt(srtFileName, translater.Histories)
}

func WriteSrt(fileName string, lines []*Line) {
	buf := bytes.NewBuffer(make([]byte, 0, len(lines)*len(lines[0].Original)))
	for i, line := range lines {
		fmt.Fprintf(buf, "%d\n", i+1)

		t := []byte(line.Original[1:31])
		t[8], t[25] = ',', ','
		buf.Write(t)
		buf.WriteByte('\n')

		buf.WriteString(line.Translated)
		buf.WriteByte('\n')
		buf.WriteByte('\n')
	}
	err := os.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
