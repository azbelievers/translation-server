package rtc

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

type ConsumerHandler interface {
	CreateConsumer(payload Payload)
	ReturnChannel() chan ReturnResult
	ListenForTrack()
}

type consumerHandler struct {
	returnChannel  chan ReturnResult
	config         webrtc.Configuration
	trackChan      chan *webrtc.TrackLocalStaticRTP
	track		*webrtc.TrackLocalStaticRTP

}

func NewConsumerHandler(trackChan chan *webrtc.TrackLocalStaticRTP, config webrtc.Configuration) ConsumerHandler {
	return &consumerHandler{
		returnChannel:  make(chan ReturnResult),
		config:         config,
		trackChan:      trackChan,
	}
}

func (r *consumerHandler) ReturnChannel() chan ReturnResult {
	return r.returnChannel
}

func (r *consumerHandler) ListenForTrack() {
	r.track = <-r.trackChan

}

func (r *consumerHandler) CreateConsumer(payload Payload) {

		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  payload.SDP,
		}

		//// Create a new RTCPeerConnection ////
		peerConnection, err := webrtc.NewPeerConnection(r.config)
		if err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}

		//defer peerConnection.Close()

		rtpSender, err := peerConnection.AddTrack(r.track)
		if err != nil {
			fmt.Println(err)
			peerConnection.Close()
		}

		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()


		fmt.Println("after Add track")

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

		peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
			println("Connection State has changed", connectionState.String())

			if connectionState == webrtc.PeerConnectionStateFailed {
				r.returnChannel <- ReturnResult{
					Offer: nil,
					Error: fmt.Errorf("connection state failed"),
				}
			}
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
