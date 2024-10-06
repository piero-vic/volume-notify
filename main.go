package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus/v5"
	"github.com/jfreymuth/pulse/proto"
)

const (
	appName         = "volume-notify"
	sinkIcon        = ""
	sinkIconMuted   = ""
	sourceIcon      = ""
	sourceIconMuted = ""
)

var (
	sinksVolumes  = make(map[string]float64)
	sourceVolumes = make(map[string]float64)
)

func main() {
	client, conn, err := proto.Connect("")
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	dbusConn, err := dbus.SessionBus()
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
	defer dbusConn.Close()

	props := proto.PropList{}
	err = client.Request(&proto.SetClientName{Props: props}, nil)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	err = client.Request(&proto.Subscribe{Mask: proto.SubscriptionMaskAll}, nil)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	eventChan := make(chan *proto.SubscribeEvent, 1)

	client.Callback = func(val interface{}) {
		event, ok := val.(*proto.SubscribeEvent)
		if !ok {
			return
		}

		if event.Event.GetType() != proto.EventChange {
			return
		}

		go func() {
			eventChan <- event
		}()
	}

	for {
		e := <-eventChan

		switch e.Event.GetFacility() {
		case proto.EventSink:
			sinkInfo, err := getSinkInfo(client, e.Index)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			if strings.Contains(sinkInfo.Device, "Monitor") {
				continue
			}

			volume := getAverageVolume(sinkInfo.ChannelVolumes)

			var body string
			if sinkInfo.Mute {
				body = fmt.Sprintf("%s  MUTED", sinkIconMuted)
				volume = -1
			} else {
				body = fmt.Sprintf("%s  Current level: %.0f%%", sinkIcon, volume)
			}

			if sinksVolumes[sinkInfo.SinkName] == volume {
				continue
			}

			sinksVolumes[sinkInfo.SinkName] = volume

			notify.SendNotification(dbusConn, notify.Notification{
				AppName:       appName,
				Summary:       fmt.Sprintf("Volume: %s", sinkInfo.Device),
				Body:          body,
				ExpireTimeout: notify.ExpireTimeoutSetByNotificationServer,
				Hints:         map[string]dbus.Variant{"value": dbus.MakeVariant(int(volume))},
			})

		case proto.EventSource:
			sourceInfo, err := getSourceInfo(client, e.Index)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			if strings.Contains(sourceInfo.Device, "Monitor") {
				continue
			}

			volume := getAverageVolume(sourceInfo.ChannelVolumes)

			var body string
			if sourceInfo.Mute {
				body = fmt.Sprintf("%s  MUTED", sourceIconMuted)
				volume = -1
			} else {
				body = fmt.Sprintf("%s  Current level: %.0f%%", sourceIcon, volume)
			}

			if sinksVolumes[sourceInfo.SourceName] == volume {
				continue
			}

			sinksVolumes[sourceInfo.SourceName] = volume

			notify.SendNotification(dbusConn, notify.Notification{
				AppName:       appName,
				Summary:       fmt.Sprintf("Volume: %s", sourceInfo.Device),
				Body:          body,
				ExpireTimeout: notify.ExpireTimeoutSetByNotificationServer,
				Hints:         map[string]dbus.Variant{"value": dbus.MakeVariant(int(volume))},
			})
		}
	}
}

func getSinkInfo(c *proto.Client, index uint32) (proto.GetSinkInfoReply, error) {
	sinkInfo := proto.GetSinkInfoReply{}
	err := c.Request(
		&proto.GetSinkInfo{SinkIndex: index},
		&sinkInfo,
	)
	return sinkInfo, err
}

func getSourceInfo(c *proto.Client, index uint32) (proto.GetSourceInfoReply, error) {
	sourceInfo := proto.GetSourceInfoReply{}
	err := c.Request(
		&proto.GetSourceInfo{SourceIndex: index},
		&sourceInfo,
	)
	return sourceInfo, err
}

func getAverageVolume(channelVolumes proto.ChannelVolumes) float64 {
	var sinkAcc int64

	for _, vol := range channelVolumes {
		sinkAcc += int64(vol)
	}
	sinkAcc /= int64(len(channelVolumes))

	return math.Ceil(float64(sinkAcc) / float64(proto.VolumeNorm) * 100.0)
}
