package rtc

import (
	"errors"
	"fmt"
	"io"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

type Payload struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

type RTCHandler interface {
	Listen(interceptor *interceptor.Registry, mediaEngine *webrtc.MediaEngine)
	ReturnChannel() chan ReturnResult
}

type ReturnResult struct {
	Offer *string
	Error error
}

type publisherHandler struct {
	channel        *chan Payload
	returnChannel  chan ReturnResult
	config         webrtc.Configuration
	localTrackChan chan *webrtc.TrackLocalStaticRTP
}

func NewPublisherHandler(channel *chan Payload, trackChan chan *webrtc.TrackLocalStaticRTP, config webrtc.Configuration) RTCHandler {
	return &publisherHandler{
		channel:        channel,
		returnChannel:  make(chan ReturnResult),
		config:         config,
		localTrackChan: trackChan,
	}
}

func (r *publisherHandler) ReturnChannel() chan ReturnResult {
	return r.returnChannel
}

func (r *publisherHandler) Listen(interceptor *interceptor.Registry, mediaEngine *webrtc.MediaEngine) {

	for {
		payload := <-*r.channel
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  payload.SDP,
		}

		//// Create a new RTCPeerConnection ////
		peerConnection, err := webrtc.NewAPI(webrtc.WithInterceptorRegistry(interceptor), webrtc.WithMediaEngine(mediaEngine)).NewPeerConnection(r.config)
		if err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}
		// Set the remote SessionDescription
		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}
		if err = peerConnection.SetLocalDescription(answer); err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}

		//// Media Handling ////

		if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}

		peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			// Create a local track, all our SFU clients will be fed via this track
			localTrack, newTrackErr := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, receiver.Track().ID(), "ABC")
			if newTrackErr != nil {
				fmt.Println(newTrackErr)
				peerConnection.Close()
			}
			r.localTrackChan <- localTrack

			rtpBuf := make([]byte, 1400)
			for {
				i, _, readErr := remoteTrack.Read(rtpBuf)
				if readErr != nil {
					fmt.Println(readErr)
					peerConnection.Close()
				}

				// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
				if _, err = localTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
					fmt.Println(err)
					peerConnection.Close()
				}
			}
		})

		peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
			println("Connection State has changed", connectionState.String())
		})

		peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				println("New ICE Candidate", candidate.String())
			}
		})

		// Sets the LocalDescription, and starts our UDP listeners
		peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
			println("ICE Gathering State has changed", state.String())
			if state == webrtc.ICEGatheringStateComplete {

				encoding := Encode(answer)

				r.returnChannel <- ReturnResult{
					Offer: &encoding,
					Error: nil,
				}
			}
		})

	}
}
