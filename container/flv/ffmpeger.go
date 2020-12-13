// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package flv

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gwuhaolin/livego/av"
	"github.com/gwuhaolin/livego/protocol/amf"
	"github.com/gwuhaolin/livego/utils/pio"
	"github.com/gwuhaolin/livego/utils/uid"
	log "github.com/sirupsen/logrus"
)

type FfmpegWriter struct {
	Uid string
	av.RWBaser
	app, title, url string
	buf             []byte
	closed          chan struct{}
	stdinAudio      io.WriteCloser
	stdinVideo      io.WriteCloser
	cmdAudio        *exec.Cmd
	cmdVideo        *exec.Cmd
}

func GetNewFfmpegWriter(info av.Info) av.WriteCloser {
	stdinAudio := io.WriteCloser(nil)
	stdinVideo := io.WriteCloser(nil)
	audioCmd := (*exec.Cmd)(nil)
	videoCmd := (*exec.Cmd)(nil)

	//ffmpegAudioCmd := []string{
	//	"ffmpeg",
	//	"-i", "pipe:0", "-map", "0:a:0", "-f", "s16le", "-acodec", "pcm_s16le",
	//	"-ac", "1", "-ar", "16000", "pipe:1",
	//	//		"-ar", "16000", "pipe:1",
	//}
	//audioCmd = exec.Command(ffmpegAudioCmd[0], ffmpegAudioCmd[1:]...)
	//stdinAudio, err := audioCmd.StdinPipe()
	//if err != nil {
	//	log.Fatalf("failed to setup StdinPipe: %v", err)
	//}
	//
	//// Uncomment to get the error output
	////audioCmd.Stderr = os.Stderr
	//
	//stdout, err := audioCmd.StdoutPipe()
	//if err != nil {
	//	log.Fatalf("failed to setup StdoutPipe: %v", err)
	//}
	//
	//// now handle the stdout
	//go func() {
	//	transcribe.StartTranscriptionStream(stdout, 30*1024)
	//}()
	//
	//if err = audioCmd.Start(); err != nil {
	//	log.Fatalf("failed to start ffmpeg audioCmd: %v", err)
	//}

	// Now do screen caps:
	filename := "warroom/frames.jpg"
	_ = os.Remove(filename)
	ffmpegVideoCmd := []string{
		"ffmpeg",
		"-i", "pipe:0", "-update", "1", "-r", "1", filename,
	}
	videoCmd = exec.Command(ffmpegVideoCmd[0], ffmpegVideoCmd[1:]...)
	stdinVideo, err := videoCmd.StdinPipe()
	if err != nil {
		log.Fatalf("failed to setup stdinVideo Pipe: %v", err)
	}
	videoCmd.Stderr = os.Stderr
	if err = videoCmd.Start(); err != nil {
		log.Fatalf("failed to start ffmpeg videoCmd: %v", err)
	}

	// Naming:
	paths := strings.SplitN(info.Key, "/", 2)
	if len(paths) != 2 {
		log.Warning("invalid info")
		return nil
	}

	writer := &FfmpegWriter{
		Uid:        uid.NewId(),
		app:        paths[0],
		title:      paths[1],
		url:        info.URL,
		stdinAudio: stdinAudio,
		stdinVideo: stdinVideo,
		cmdAudio:   audioCmd,
		cmdVideo:   videoCmd,
		RWBaser:    av.NewRWBaser(time.Second * 10),
		closed:     make(chan struct{}),
		buf:        make([]byte, headerLen),
	}

	if writer.stdinAudio != nil {
		_, err = writer.stdinAudio.Write(flvHeader)
		if err != nil {
			log.Errorf("error writing to stdinAudio: %v", err)
		}
	}
	if writer.stdinVideo != nil {
		_, err = writer.stdinVideo.Write(flvHeader)
		if err != nil {
			log.Errorf("error writing to stdinAudio: %v", err)
		}
	}

	pio.PutI32BE(writer.buf[:4], 0)

	if writer.stdinAudio != nil {
		_, err = writer.stdinAudio.Write(writer.buf[:4])
		if err != nil {
			log.Errorf("error writing to stdinAudio: %v", err)
		}
	}

	if writer.stdinVideo != nil {
		_, err = writer.stdinVideo.Write(writer.buf[:4])
		if err != nil {
			log.Errorf("error writing to stdinAudio: %v", err)
		}
	}
	log.Warn("new FfmpegWriter: ", writer.Info())
	return writer
}

func (writer *FfmpegWriter) Write(p *av.Packet) error {
	writer.RWBaser.SetPreTime()
	h := writer.buf[:headerLen]
	typeID := av.TAG_VIDEO
	if !p.IsVideo {
		if p.IsMetadata {
			var err error
			typeID = av.TAG_SCRIPTDATAAMF0
			p.Data, err = amf.MetaDataReform(p.Data, amf.DEL)
			if err != nil {
				return err
			}
		} else {
			typeID = av.TAG_AUDIO
		}
	}
	dataLen := len(p.Data)
	timestamp := p.TimeStamp
	timestamp += writer.BaseTimeStamp()
	writer.RWBaser.RecTimeStamp(timestamp, uint32(typeID))

	preDataLen := dataLen + headerLen
	timestampbase := timestamp & 0xffffff
	timestampExt := timestamp >> 24 & 0xff

	pio.PutU8(h[0:1], uint8(typeID))
	pio.PutI24BE(h[1:4], int32(dataLen))
	pio.PutI24BE(h[4:7], int32(timestampbase))
	pio.PutU8(h[7:8], uint8(timestampExt))

	if writer.stdinAudio != nil {
		if _, err := writer.stdinAudio.Write(h); err != nil {
			return err
		}
		if _, err := writer.stdinAudio.Write(p.Data); err != nil {
			return err
		}
	}
	if writer.stdinVideo != nil {
		if _, err := writer.stdinVideo.Write(h); err != nil {
			return err
		}
		if _, err := writer.stdinVideo.Write(p.Data); err != nil {
			return err
		}
	}

	pio.PutI32BE(h[:4], int32(preDataLen))

	if writer.stdinAudio != nil {
		if _, err := writer.stdinAudio.Write(h[:4]); err != nil {
			return err
		}
	}
	if writer.stdinVideo != nil {
		if _, err := writer.stdinVideo.Write(h[:4]); err != nil {
			return err
		}
	}
	return nil
}

func (writer *FfmpegWriter) Wait() {
	select {
	case <-writer.closed:
		return
	}
}

func (writer *FfmpegWriter) Close(err error) {
	if writer.stdinAudio != nil {
		if err := writer.stdinAudio.Close(); err != nil {
			log.Errorf("error closing FfmpegWriter out: %v", err)
		}
		if err := writer.cmdAudio.Wait(); err != nil {
			log.Errorf("error closing FfmpegWriter cmdAudio: %v", err)
		}
	}
	if writer.stdinVideo != nil {
		if err := writer.stdinVideo.Close(); err != nil {
			log.Errorf("error closing FfmpegWriter out: %v", err)
		}
		if err := writer.cmdVideo.Wait(); err != nil {
			log.Errorf("error closing FfmpegWriter cmdVideo: %v", err)
		}
	}

	close(writer.closed)
	log.Warnf("Closed FfmpegWriter %s with error: %v", writer.Info().Key, err)
}

func (writer *FfmpegWriter) Info() (ret av.Info) {
	ret.UID = writer.Uid
	ret.URL = writer.url
	ret.Key = writer.app + "/" + writer.title
	return
}
