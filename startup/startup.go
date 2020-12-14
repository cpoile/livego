// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package startup

import (
	"github.com/cpoile/livego/configure"
	"github.com/cpoile/livego/rtmpservice"
	log "github.com/sirupsen/logrus"
	"net"
)

func MyStartRtmp(stream *rtmpservice.RtmpStream) {
	rtmpAddr := configure.Config.GetString("rtmp_addr")

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		log.Fatal(err)
	}

	rtmpServer := rtmpservice.NewRtmpServer(stream)

	defer func() {
		if r := recover(); r != nil {
			log.Error("RTMP server panic: ", r)
		}
	}()
	log.Info("RTMP Listen On ", rtmpAddr)
	rtmpServer.Serve(rtmpListen)
}
