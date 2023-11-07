import { Dispatch, RefObject, SetStateAction } from "react";

export default class RtcHandler {
  private peerConnection = new RTCPeerConnection({
    iceServers: [{ urls: "stun:stun.stunprotocol.org" }],
    iceTransportPolicy: "all",
  });

  public async listenForDeviceChanges(
    stateUpdateFn: Dispatch<SetStateAction<MediaDeviceInfo[]>>
  ) {
    stateUpdateFn(await this.listAudioDevices());
    navigator.mediaDevices.addEventListener("devicechange", async () => {
      // on event, refetch device list and push to state with the provided setstate fn.
      const devices = await this.listAudioDevices();
      stateUpdateFn(devices);
    });
  }

  public async OpenDeviceStream(id: string): Promise<MediaStream> {
    return await navigator.mediaDevices.getUserMedia({
      audio: { deviceId: id },
    });
  }

  private async listAudioDevices() {
    const devices = await navigator.mediaDevices.enumerateDevices();
    return devices.filter((d) => d.kind === "audioinput");
  }

  ///

  public async connectPublisher(deviceId?: string) {
    const offer = await this.peerConnection.createOffer();
    await this.peerConnection.setLocalDescription(offer);

    // add track
    const track = await this.OpenDeviceStream(deviceId || "default");
    this.peerConnection.addTrack(track.getTracks()[0]);

    // listeners
    this.peerConnection.onconnectionstatechange = () => {
      console.log(this.peerConnection.connectionState);
    };

    this.peerConnection.onicecandidate = (e) => {
      if (!e.candidate) return;
      console.log("ICE Candidate" + JSON.stringify(e.candidate));
    };

    this.peerConnection.onicegatheringstatechange = async () => {
      console.log("ICE Gathering State", this.peerConnection.iceGatheringState);

      if (this.peerConnection.iceGatheringState === "complete") {
        console.log("ICE Gathering Complete");

        const remoteDesc = await fetch("/publisher", {
          method: "POST",
          body: JSON.stringify(this.peerConnection.localDescription),
        });

        const remoteDescJson = await remoteDesc.json();
        const remoteDescObj = JSON.parse(
          atob(remoteDescJson.offer)
        ) as RTCSessionDescriptionInit;

        await this.peerConnection
          .setRemoteDescription(new RTCSessionDescription(remoteDescObj))
          .catch((e) => console.log(e));
      }
    };
  }

  public async connectSubscriber(ref: RefObject<HTMLAudioElement>) {
    const offer = await this.peerConnection.createOffer();
    await this.peerConnection.setLocalDescription(offer);

    // listeners
    this.peerConnection.onconnectionstatechange = () => {
      console.log("Connection state: " + this.peerConnection.connectionState);
    };

    this.peerConnection.onicecandidate = (e) => {
      if (!e.candidate) return;
      console.log("ICE Candidate: " + JSON.stringify(e.candidate));
    };

    this.peerConnection.onicegatheringstatechange = async () => {
      console.log(
        "ICE Gathering State: ",
        this.peerConnection.iceGatheringState
      );

      if (this.peerConnection.iceGatheringState === "complete") {
        const remoteDesc = await fetch("/consumer", {
          method: "POST",
          body: JSON.stringify(this.peerConnection.localDescription),
        });

        const remoteDescJson = await remoteDesc.json();
        const remoteDescObj = JSON.parse(
          atob(remoteDescJson.offer)
        ) as RTCSessionDescriptionInit;

        this.peerConnection.ontrack = (event) => {
          if (!ref.current) return;
          ref.current.srcObject = event.streams[0];
        };

        await this.peerConnection
          .setRemoteDescription(new RTCSessionDescription(remoteDescObj))
          .catch((e) => console.log(e));
      }
    };
  }
}
